package web

import (
	"context"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"modbus-energy-meter-sim/pkg/meter"
	"modbus-energy-meter-sim/pkg/modbus"
)

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server provides the embedded web dashboard and WebSocket streaming endpoint.
type Server struct {
	addr      string
	dm        *meter.DeviceManager
	rtuServer *modbus.RTUServer
	tcpServer *modbus.TCPServer

	clientsMu sync.Mutex
	clients   map[*websocket.Conn]bool
}

// NewServer creates a new Web server.
func NewServer(addr string, dm *meter.DeviceManager, rtuServer *modbus.RTUServer, tcpServer *modbus.TCPServer) *Server {
	s := &Server{
		addr:      addr,
		dm:        dm,
		rtuServer: rtuServer,
		tcpServer: tcpServer,
		clients:   make(map[*websocket.Conn]bool),
	}

	if rtuServer != nil {
		rtuServer.SubscribePacketLogs(func(p *meter.PacketLog) {
			s.broadcastPacket(p)
		})
	}

	return s
}

// Start launches the HTTP and WebSocket server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static assets
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// API endpoints
	mux.HandleFunc("/api/meters", s.handleMeters)
	mux.HandleFunc("/api/meters/add", s.handleAddMeter)
	mux.HandleFunc("/api/meters/remove", s.handleRemoveMeter)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/control", s.handleControl)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/ports", s.handlePorts)
	mux.HandleFunc("/api/serial/status", s.handleSerialStatus)
	mux.HandleFunc("/api/serial/connect", s.handleSerialConnect)
	mux.HandleFunc("/api/serial/disconnect", s.handleSerialDisconnect)
	mux.HandleFunc("/api/tcp/status", s.handleTCPStatus)
	mux.HandleFunc("/api/tcp/update", s.handleTCPUpdate)
	mux.HandleFunc("/ws", s.handleWebSocket)

	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("[Web] Dashboard & Reader ready at http://localhost%s", s.addr)

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleMeters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.dm.GetAllStates())
}

func (s *Server) handleAddMeter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID   uint8  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m, err := s.dm.AddMeter(req.ID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.GetState())
}

func (s *Server) handleRemoveMeter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID uint8 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.dm.RemoveMeter(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			rm := s.dm.GetMeter(uint8(id))
			if rm != nil {
				json.NewEncoder(w).Encode(rm.GetState())
				return
			}
		}
	}
	json.NewEncoder(w).Encode(s.dm.GetDefaultMeter().GetState())
}

func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	ports, err := modbus.ListAvailablePorts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ports": ports})
}

func (s *Server) handleSerialStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	connected, port, baud := false, "", 9600
	if s.rtuServer != nil {
		connected, port, baud = s.rtuServer.GetStatus()
	}

	tcpActive, tcpAddr, tcpErr := false, ":8502", ""
	if s.tcpServer != nil {
		tcpActive, tcpAddr, tcpErr = s.tcpServer.GetStatus()
	}

	json.NewEncoder(w).Encode(map[string]any{
		"connected":  connected,
		"port":       port,
		"baud":       baud,
		"tcp_active": tcpActive,
		"tcp_addr":   tcpAddr,
		"tcp_err":    tcpErr,
	})
}

func (s *Server) handleTCPStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.tcpServer == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"active": false,
			"addr":   "disabled",
			"error":  "TCP server not initialized",
		})
		return
	}
	active, addr, lastErr := s.tcpServer.GetStatus()
	json.NewEncoder(w).Encode(map[string]any{
		"active": active,
		"addr":   addr,
		"error":  lastErr,
	})
}

func (s *Server) handleTCPUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.tcpServer == nil {
		http.Error(w, "TCP server not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Port string `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.tcpServer.ListenOn(req.Port); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	active, addr, lastErr := s.tcpServer.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"active":  active,
		"addr":    addr,
		"error":   lastErr,
	})
}

func (s *Server) handleSerialConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Port string `json:"port"`
		Baud int    `json:"baud"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.rtuServer == nil {
		http.Error(w, "RTU server not initialized", http.StatusInternalServerError)
		return
	}

	if err := s.rtuServer.Connect(req.Port, req.Baud); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleSerialStatus(w, r)
}

func (s *Server) handleSerialDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.rtuServer != nil {
		_ = s.rtuServer.Disconnect()
	}
	s.handleSerialStatus(w, r)
}

func (s *Server) handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MeterID           uint8    `json:"meter_id"`
		Name              *string  `json:"name"`
		Voltage           *float64 `json:"target_voltage_v"`
		Current           *float64 `json:"target_current_a"`
		PowerFactor       *float64 `json:"target_pf"`
		Frequency         *float64 `json:"target_frequency_hz"`
		RelayState        *bool    `json:"relay_state"`
		SimulateNoise     *bool    `json:"simulate_noise"`
		DynamicEnergy     *bool    `json:"dynamic_energy"`
		ResetEnergy       bool     `json:"reset_energy"`
		StationAddress    *uint8   `json:"station_address"`
		BaudRateCode      *uint8   `json:"baud_rate_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rm := s.dm.GetMeter(req.MeterID)
	if rm == nil {
		rm = s.dm.GetDefaultMeter()
	}

	updated := rm.UpdateState(func(state *meter.MeterState) {
		if req.Name != nil && *req.Name != "" {
			state.Name = *req.Name
		}
		if req.Voltage != nil {
			state.TargetVoltage = *req.Voltage
		}
		if req.Current != nil {
			state.TargetCurrent = *req.Current
		}
		if req.PowerFactor != nil {
			state.TargetPowerFactor = *req.PowerFactor
		}
		if req.Frequency != nil {
			state.TargetFrequency = *req.Frequency
		}
		if req.RelayState != nil {
			state.RelayState = *req.RelayState
		}
		if req.SimulateNoise != nil {
			state.SimulateNoise = *req.SimulateNoise
		}
		if req.DynamicEnergy != nil {
			state.DynamicEnergy = *req.DynamicEnergy
		}
		if req.ResetEnergy {
			state.TotalEnergy = 0
			state.ExportEnergy = 0
			state.ImportEnergy = 0
		}
		if req.StationAddress != nil && *req.StationAddress > 0 {
			state.StationAddress = *req.StationAddress
		}
		if req.BaudRateCode != nil && *req.BaudRateCode >= 1 && *req.BaudRateCode <= 4 {
			state.BaudRateCode = *req.BaudRateCode
		}
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		HexCommand string `json:"hex_command"`
		Preset     string `json:"preset"`
		SlaveID    uint8  `json:"slave_id"`
		StartAddr  uint16 `json:"start_addr"`
		RegCount   uint16 `json:"reg_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SlaveID == 0 {
		req.SlaveID = s.dm.GetDefaultMeter().GetState().StationAddress
	}

	rm := s.dm.GetMeter(req.SlaveID)
	if rm == nil {
		rm = s.dm.GetDefaultMeter()
	}

	var rawBytes []byte

	if req.HexCommand != "" {
		cleaned := strings.ReplaceAll(strings.ReplaceAll(req.HexCommand, " ", ""), "0x", "")
		var err error
		rawBytes, err = hex.DecodeString(cleaned)
		if err != nil {
			http.Error(w, "Invalid hex string: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(rawBytes) >= 4 && !modbus.CheckCRC(rawBytes) {
			rawBytes = modbus.AppendCRC(rawBytes)
		}
	} else if req.Preset != "" {
		switch req.Preset {
		case "voltage":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegVoltage, 1)
		case "current":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegCurrent, 1)
		case "power_factor":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegPowerFactor, 1)
		case "frequency":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegFrequency, 1)
		case "power_all":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegVoltage, 6)
		case "energy":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegTotalEnergyHigh, 2)
		case "export_energy":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegExportEnergyHigh, 2)
		case "import_energy":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegImportEnergyHigh, 2)
		case "energy_all":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegTotalEnergyHigh, 12)
		case "relay_on":
			rawBytes = modbus.BuildWriteMultipleRequest(req.SlaveID, meter.RegRelay, []uint16{1})
		case "relay_off":
			rawBytes = modbus.BuildWriteMultipleRequest(req.SlaveID, meter.RegRelay, []uint16{0})
		case "reset_energy":
			rawBytes = modbus.BuildWriteMultipleRequest(req.SlaveID, meter.RegTotalEnergyHigh, []uint16{0, 0})
		case "address_baud":
			rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, meter.RegAddressBaud, 1)
		default:
			http.Error(w, "Unknown preset: "+req.Preset, http.StatusBadRequest)
			return
		}
	} else {
		if req.RegCount == 0 {
			req.RegCount = 1
		}
		rawBytes = modbus.BuildReadHoldingRequest(req.SlaveID, req.StartAddr, req.RegCount)
	}

	result := modbus.ExecuteInternalQuery(rm, rawBytes)

	if s.rtuServer != nil {
		s.rtuServer.BroadcastLog(&meter.PacketLog{
			Timestamp:   time.Now(),
			Direction:   "RX (Web Reader -> Meter)",
			RawHex:      result.RequestHex,
			Length:      len(rawBytes),
			SlaveID:     req.SlaveID,
			DeviceName:  fmt.Sprintf("Meter #%d (%s)", req.SlaveID, rm.GetState().Name),
			Function:    rawBytes[1],
			CRCValid:    true,
			Description: fmt.Sprintf("Web Reader Query (Func 0x%02X)", rawBytes[1]),
			ByteDetails: result.RequestDetails,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Web] WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	initMsg, _ := json.Marshal(map[string]any{
		"type": "meters",
		"data": s.dm.GetAllStates(),
	})
	conn.WriteMessage(websocket.TextMessage, initMsg)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data, _ := json.Marshal(map[string]any{
				"type": "meters",
				"data": s.dm.GetAllStates(),
			})
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

func (s *Server) broadcastPacket(p *meter.PacketLog) {
	msg, err := json.Marshal(map[string]any{
		"type":   "packet",
		"packet": p,
	})
	if err != nil {
		return
	}

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for client := range s.clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}

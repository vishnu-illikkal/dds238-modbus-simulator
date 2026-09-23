package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"modbus-energy-meter-sim/pkg/meter"
)

// TCPServer implements a standard Modbus TCP server (as well as raw RTU-over-TCP).
type TCPServer struct {
	mu        sync.Mutex
	addr      string
	listener  net.Listener
	dm        *meter.DeviceManager
	rtuServer *RTUServer
	ctx       context.Context
	cancel    context.CancelFunc
	active    bool
	lastErr   string
}

// NewTCPServer creates a new Modbus TCP server.
func NewTCPServer(addr string, dm *meter.DeviceManager, rtuServer *RTUServer) *TCPServer {
	return &TCPServer{
		addr:      addr,
		dm:        dm,
		rtuServer: rtuServer,
	}
}

// Start launches the initial Modbus TCP listener.
func (s *TCPServer) Start(parentCtx context.Context) error {
	return s.ListenOn(s.addr, parentCtx)
}

// ListenOn binds to a new TCP address/port dynamically.
func (s *TCPServer) ListenOn(newAddr string, parentCtx ...context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Normalize port address (e.g. "8502" -> ":8502")
	trimmed := strings.TrimSpace(newAddr)
	if trimmed == "" || strings.EqualFold(trimmed, "off") || strings.EqualFold(trimmed, "disabled") {
		s.stopLocked()
		s.addr = "disabled"
		s.active = false
		s.lastErr = ""
		log.Printf("[TCP] Modbus TCP server disabled.")
		return nil
	}

	if !strings.Contains(trimmed, ":") {
		trimmed = ":" + trimmed
	}

	// Stop previous listener if active
	s.stopLocked()

	var baseCtx context.Context
	if len(parentCtx) > 0 && parentCtx[0] != nil {
		baseCtx = parentCtx[0]
	} else if s.ctx != nil {
		baseCtx = context.Background()
	} else {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithCancel(baseCtx)

	listener, err := net.Listen("tcp", trimmed)
	if err != nil {
		cancel()
		s.addr = trimmed
		s.active = false
		s.lastErr = err.Error()
		return fmt.Errorf("failed to bind TCP address %s: %w", trimmed, err)
	}

	s.addr = trimmed
	s.listener = listener
	s.ctx = ctx
	s.cancel = cancel
	s.active = true
	s.lastErr = ""

	log.Printf("[TCP] Modbus TCP server listening on %s...", trimmed)

	go func(l net.Listener, c context.Context) {
		for {
			conn, err := l.Accept()
			if err != nil {
				if c.Err() != nil {
					return
				}
				return
			}
			go s.handleConnection(c, conn)
		}
	}(listener, ctx)

	return nil
}

func (s *TCPServer) stopLocked() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
	s.active = false
}

// Stop stops the TCP server.
func (s *TCPServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

// GetStatus returns the current status, address, and any last error.
func (s *TCPServer) GetStatus() (active bool, addr string, lastErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active, s.addr, s.lastErr
}

func (s *TCPServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 512)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				// Normal disconnect
			}
			return
		}

		if n < 4 {
			continue
		}

		// Check if this is Modbus TCP MBAP Header (Protocol ID at bytes 2-3 == 0x0000 and len >= 7)
		if n >= 7 && binary.BigEndian.Uint16(buf[2:4]) == 0 {
			txBytes := s.processMBAPFrame(buf[:n])
			if len(txBytes) > 0 {
				conn.Write(txBytes)
			}
		} else {
			// Handle as raw RTU frame over TCP
			var counter int64
			txBytes, rxLog, txLog := ProcessRTUFrame(buf[:n], s.dm, &counter)
			if rxLog != nil && s.rtuServer != nil {
				s.rtuServer.BroadcastLog(rxLog)
			}
			if len(txBytes) > 0 {
				conn.Write(txBytes)
				if txLog != nil && s.rtuServer != nil {
					s.rtuServer.BroadcastLog(txLog)
				}
			}
		}
	}
}

func (s *TCPServer) processMBAPFrame(req []byte) []byte {
	transID := binary.BigEndian.Uint16(req[0:2])
	protoID := binary.BigEndian.Uint16(req[2:4])
	pduLen := binary.BigEndian.Uint16(req[4:6])
	unitID := req[6]

	if int(pduLen)+6 > len(req) || protoID != 0 {
		return nil
	}

	funcCode := req[7]
	rm := s.dm.GetMeter(unitID)
	if rm == nil {
		if unitID == 0 || unitID == 255 {
			rm = s.dm.GetDefaultMeter()
		} else {
			return nil
		}
	}

	var respPDU []byte

	switch funcCode {
	case FuncReadHoldingRegisters:
		if len(req) < 12 {
			respPDU = []byte{funcCode | 0x80, meter.ExpIllegalDataValue}
		} else {
			startAddr := binary.BigEndian.Uint16(req[8:10])
			count := binary.BigEndian.Uint16(req[10:12])
			data, exp := rm.ReadHoldingRegisters(startAddr, count)
			if exp != 0 {
				respPDU = []byte{funcCode | 0x80, exp}
			} else {
				respPDU = []byte{funcCode, byte(len(data))}
				respPDU = append(respPDU, data...)
			}
		}

	case FuncWriteMultipleRegisters:
		if len(req) < 13 {
			respPDU = []byte{funcCode | 0x80, meter.ExpIllegalDataValue}
		} else {
			startAddr := binary.BigEndian.Uint16(req[8:10])
			count := binary.BigEndian.Uint16(req[10:12])
			byteCount := req[12]
			if len(req) < 13+int(byteCount) {
				respPDU = []byte{funcCode | 0x80, meter.ExpIllegalDataValue}
			} else {
				data := req[13 : 13+byteCount]
				exp := rm.WriteMultipleRegisters(startAddr, count, data)
				if exp != 0 {
					respPDU = []byte{funcCode | 0x80, exp}
				} else {
					respPDU = []byte{funcCode, req[8], req[9], req[10], req[11]}
				}
			}
		}

	case FuncWriteSingleRegister:
		respPDU = []byte{funcCode | 0x80, meter.ExpIllegalFunction}

	default:
		respPDU = []byte{funcCode | 0x80, meter.ExpIllegalFunction}
	}

	respHeader := make([]byte, 7)
	binary.BigEndian.PutUint16(respHeader[0:2], transID)
	binary.BigEndian.PutUint16(respHeader[2:4], 0) // Protocol ID
	binary.BigEndian.PutUint16(respHeader[4:6], uint16(1+len(respPDU)))
	respHeader[6] = unitID

	resp := append(respHeader, respPDU...)

	// Create packet logs for UI
	if s.rtuServer != nil {
		s.rtuServer.BroadcastLog(&meter.PacketLog{
			Timestamp:   time.Now(),
			Direction:   "RX (TCP Client -> Meter)",
			RawHex:      FormatHex(req),
			Length:      len(req),
			SlaveID:     unitID,
			Function:    funcCode,
			CRCValid:    true,
			Description: fmt.Sprintf("TCP Modbus Request (Func 0x%02X)", funcCode),
		})
		s.rtuServer.BroadcastLog(&meter.PacketLog{
			Timestamp:   time.Now(),
			Direction:   "TX (Meter -> TCP Client)",
			RawHex:      FormatHex(resp),
			Length:      len(resp),
			SlaveID:     unitID,
			Function:    respPDU[0],
			CRCValid:    true,
			Description: fmt.Sprintf("TCP Modbus Response (Len %d)", len(resp)),
		})
	}

	return resp
}

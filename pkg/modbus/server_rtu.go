package modbus

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
	"modbus-energy-meter-sim/pkg/meter"
)

// LogSubscriber is a function that receives new packet log entries.
type LogSubscriber func(p *meter.PacketLog)

// RTUServer manages the dynamic Modbus RTU Serial connection.
type RTUServer struct {
	dm            *meter.DeviceManager
	packetCounter int64

	subscribersMu sync.RWMutex
	subscribers   []LogSubscriber

	portMu      sync.Mutex
	port        serial.Port
	portName    string
	baudRate    int
	isConnected bool
	cancelFunc  context.CancelFunc
}

// NewRTUServer creates a new RTUServer instance.
func NewRTUServer(dm *meter.DeviceManager) *RTUServer {
	return &RTUServer{
		dm:       dm,
		baudRate: 9600,
	}
}

// SubscribePacketLogs registers a listener callback for all Modbus packets.
func (s *RTUServer) SubscribePacketLogs(sub LogSubscriber) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	s.subscribers = append(s.subscribers, sub)
}

// BroadcastLog notifies all subscribers of a new packet.
func (s *RTUServer) BroadcastLog(p *meter.PacketLog) {
	if p == nil {
		return
	}
	s.subscribersMu.RLock()
	subs := append([]LogSubscriber(nil), s.subscribers...)
	s.subscribersMu.RUnlock()

	for _, sub := range subs {
		sub(p)
	}
}

// ListAvailablePorts returns all serial/COM ports detected on the system.
func ListAvailablePorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	return ports, nil
}

// GetStatus returns the current connection state.
func (s *RTUServer) GetStatus() (bool, string, int) {
	s.portMu.Lock()
	defer s.portMu.Unlock()
	return s.isConnected, s.portName, s.baudRate
}

// Connect opens the specified serial port dynamically.
func (s *RTUServer) Connect(portName string, baudRate int) error {
	s.portMu.Lock()
	defer s.portMu.Unlock()

	if s.isConnected {
		// Close existing connection first
		if s.cancelFunc != nil {
			s.cancelFunc()
		}
		if s.port != nil {
			s.port.Close()
			s.port = nil
		}
		s.isConnected = false
	}

	if portName == "" {
		return fmt.Errorf("no port specified")
	}
	if baudRate <= 0 {
		baudRate = 9600
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %w", portName, err)
	}

	// 20ms read timeout for Modbus RTU inter-frame silence detection
	_ = port.SetReadTimeout(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	s.port = port
	s.portName = portName
	s.baudRate = baudRate
	s.isConnected = true
	s.cancelFunc = cancel

	log.Printf("[RTU] Successfully connected to %s at %d Bd (8N1)", portName, baudRate)

	go s.readLoop(ctx, port, portName)
	return nil
}

// Disconnect closes the active serial port.
func (s *RTUServer) Disconnect() error {
	s.portMu.Lock()
	defer s.portMu.Unlock()

	if !s.isConnected {
		return nil
	}

	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
	if s.port != nil {
		s.port.Close()
		s.port = nil
	}
	s.isConnected = false
	log.Printf("[RTU] Disconnected from %s", s.portName)
	return nil
}

func (s *RTUServer) readLoop(ctx context.Context, port serial.Port, portName string) {
	buf := make([]byte, 512)
	frameBuf := make([]byte, 0, 512)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := port.Read(buf)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				log.Printf("[RTU] Read error on %s: %v", portName, err)
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}

		if n > 0 {
			frameBuf = append(frameBuf, buf[:n]...)
			// Keep reading until silence timeout
			for {
				nMore, err := port.Read(buf)
				if err != nil || nMore == 0 {
					break
				}
				frameBuf = append(frameBuf, buf[:nMore]...)
			}

			// Process complete RTU frame
			if len(frameBuf) >= 4 {
				respBytes, rxLog, txLog := ProcessRTUFrame(frameBuf, s.dm, &s.packetCounter)
				if rxLog != nil {
					s.BroadcastLog(rxLog)
					log.Printf("[RTU <- STM32] %s: %s", rxLog.RawHex, rxLog.Description)
				}

				if len(respBytes) > 0 {
					// 5ms turnaround time
					time.Sleep(5 * time.Millisecond)
					_, writeErr := port.Write(respBytes)
					if writeErr != nil {
						log.Printf("[RTU] Write error: %v", writeErr)
					} else {
						if txLog != nil {
							s.BroadcastLog(txLog)
							log.Printf("[RTU -> STM32] %s: %s", txLog.RawHex, txLog.Description)
						}
					}
				}
			}

			frameBuf = frameBuf[:0]
		}
	}
}

package meter

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"
)

// Modbus Exception Codes
const (
	ExpIllegalFunction    byte = 0x01
	ExpIllegalDataAddress byte = 0x02
	ExpIllegalDataValue   byte = 0x03
	ExpSlaveDeviceFailure byte = 0x04
)

// Register Addresses
const (
	RegTotalEnergyHigh  = 0x0000 // 1/100 kWh
	RegTotalEnergyLow   = 0x0001
	RegExportEnergyHigh = 0x0008 // 1/100 kWh
	RegExportEnergyLow  = 0x0009
	RegImportEnergyHigh = 0x000A // 1/100 kWh
	RegImportEnergyLow  = 0x000B
	RegVoltage          = 0x000C // 1/10 V
	RegCurrent          = 0x000D // 1/100 A
	RegActivePower      = 0x000E // 1 W (signed)
	RegReactivePower    = 0x000F // 1 VAr
	RegPowerFactor      = 0x0010 // 1/1000
	RegFrequency        = 0x0011 // 1/100 Hz
	RegAddressBaud      = 0x0015 // High: Station Addr (1-247), Low: Baud Code (1-4)
	RegRelay            = 0x001A // 0=Off, 1=On
	MaxRegisterAddress  = 0x001A
)

// RegisterManager handles thread-safe reading and writing of simulated DDS238-2 registers.
type RegisterManager struct {
	mu    sync.RWMutex
	state MeterState
}

// NewRegisterManager creates a RegisterManager initialized with default realistic values.
func NewRegisterManager(slaveID uint8) *RegisterManager {
	return NewNamedRegisterManager(slaveID, fmt.Sprintf("Meter #%d", slaveID))
}

// NewNamedRegisterManager creates a RegisterManager with a specific name/role.
func NewNamedRegisterManager(slaveID uint8, name string) *RegisterManager {
	if slaveID == 0 {
		slaveID = 1
	}
	if name == "" {
		name = fmt.Sprintf("Meter #%d", slaveID)
	}

	rm := &RegisterManager{
		state: MeterState{
			Name:              name,
			TotalEnergy:       124.56,
			ExportEnergy:      0.0,
			ImportEnergy:      124.56,
			Voltage:           230.4,
			Current:           5.25,
			ActivePower:       1209,
			ReactivePower:     115,
			PowerFactor:       0.995,
			Frequency:         50.02,
			StationAddress:    slaveID,
			BaudRateCode:      1, // 9600 Bd
			RelayState:        true,
			SimulateNoise:     true,
			DynamicEnergy:     true,
			TargetVoltage:     230.0,
			TargetCurrent:     5.25,
			TargetPowerFactor: 0.995,
			TargetFrequency:   50.00,
			LastUpdated:       time.Now(),
		},
	}
	return rm
}

// GetState returns a snapshot copy of current meter state.
func (rm *RegisterManager) GetState() MeterState {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.state
}

// UpdateState allows updating meter configuration/state safely.
func (rm *RegisterManager) UpdateState(updater func(s *MeterState)) MeterState {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	updater(&rm.state)
	rm.state.LastUpdated = time.Now()
	return rm.state
}

// BuildRegisterTable constructs the 16-bit register map array (0x0000 to 0x001A).
func (rm *RegisterManager) buildRegisterTableLocked() []uint16 {
	table := make([]uint16, MaxRegisterAddress+1)

	// Total Energy (uint32, 0.01 kWh)
	totalUnits := uint32(math.Round(rm.state.TotalEnergy * 100))
	table[RegTotalEnergyHigh] = uint16(totalUnits >> 16)
	table[RegTotalEnergyLow] = uint16(totalUnits & 0xFFFF)

	// Export Energy (uint32, 0.01 kWh)
	exportUnits := uint32(math.Round(rm.state.ExportEnergy * 100))
	table[RegExportEnergyHigh] = uint16(exportUnits >> 16)
	table[RegExportEnergyLow] = uint16(exportUnits & 0xFFFF)

	// Import Energy (uint32, 0.01 kWh)
	importUnits := uint32(math.Round(rm.state.ImportEnergy * 100))
	table[RegImportEnergyHigh] = uint16(importUnits >> 16)
	table[RegImportEnergyLow] = uint16(importUnits & 0xFFFF)

	// Voltage (0.1 V)
	table[RegVoltage] = uint16(math.Round(rm.state.Voltage * 10))

	// Current (0.01 A)
	table[RegCurrent] = uint16(math.Round(rm.state.Current * 100))

	// Active Power (1 W signed)
	table[RegActivePower] = uint16(int16(math.Round(rm.state.ActivePower)))

	// Reactive Power (1 VAr)
	table[RegReactivePower] = uint16(math.Round(math.Abs(rm.state.ReactivePower)))

	// Power Factor (0.001)
	table[RegPowerFactor] = uint16(math.Round(rm.state.PowerFactor * 1000))

	// Frequency (0.01 Hz)
	table[RegFrequency] = uint16(math.Round(rm.state.Frequency * 100))

	// Station Address (High Byte) & Baud Rate (Low Byte)
	table[RegAddressBaud] = (uint16(rm.state.StationAddress) << 8) | uint16(rm.state.BaudRateCode)

	// Relay (0x0000 or 0x0001)
	if rm.state.RelayState {
		table[RegRelay] = 0x0001
	} else {
		table[RegRelay] = 0x0000
	}

	return table
}

// ReadHoldingRegisters reads count registers starting from startAddr.
func (rm *RegisterManager) ReadHoldingRegisters(startAddr uint16, count uint16) ([]byte, byte) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if count < 1 || count > 125 {
		return nil, ExpIllegalDataValue
	}

	endAddr := startAddr + count - 1
	if endAddr > MaxRegisterAddress {
		return nil, ExpIllegalDataAddress
	}

	table := rm.buildRegisterTableLocked()
	buf := make([]byte, count*2)
	for i := uint16(0); i < count; i++ {
		regVal := table[startAddr+i]
		binary.BigEndian.PutUint16(buf[i*2:(i+1)*2], regVal)
	}

	return buf, 0
}

// WriteMultipleRegisters handles writing multiple registers (Function 0x10).
func (rm *RegisterManager) WriteMultipleRegisters(startAddr uint16, count uint16, data []byte) byte {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if count < 1 || count > 123 || len(data) != int(count*2) {
		return ExpIllegalDataValue
	}

	endAddr := startAddr + count - 1
	if endAddr > MaxRegisterAddress {
		return ExpIllegalDataAddress
	}

	for i := uint16(0); i < count; i++ {
		addr := startAddr + i
		val := binary.BigEndian.Uint16(data[i*2 : (i+1)*2])

		switch addr {
		case RegTotalEnergyHigh, RegTotalEnergyLow:
			// Per Note 1: Total, export and import energy counters can be erased writing 0 in total energy registers
			if count >= 2 && startAddr == RegTotalEnergyHigh {
				high := binary.BigEndian.Uint16(data[0:2])
				low := binary.BigEndian.Uint16(data[2:4])
				if high == 0 && low == 0 {
					rm.state.TotalEnergy = 0
					rm.state.ExportEnergy = 0
					rm.state.ImportEnergy = 0
				}
			}
		case RegAddressBaud:
			// High byte: station address (1-247), Low byte: baud rate (1-4)
			addrByte := uint8(val >> 8)
			baudByte := uint8(val & 0xFF)
			if addrByte >= 1 && addrByte <= 247 {
				rm.state.StationAddress = addrByte
			}
			if baudByte >= 1 && baudByte <= 4 {
				rm.state.BaudRateCode = baudByte
			}
		case RegRelay:
			// Note 3: 0 = Off, 1 = On
			rm.state.RelayState = (val != 0)
		}
	}

	rm.state.LastUpdated = time.Now()
	return 0
}

// BaudRateFromCode converts 1-4 code to integer baud rate.
func BaudRateFromCode(code uint8) int {
	switch code {
	case 1:
		return 9600
	case 2:
		return 4800
	case 3:
		return 2400
	case 4:
		return 1200
	default:
		return 9600
	}
}

// FormatRegisterMap returns a human-readable string of the current register values.
func (rm *RegisterManager) FormatRegisterMap() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return fmt.Sprintf("V: %.1fV | I: %.2fA | P: %.0fW | PF: %.3f | F: %.2fHz | E: %.2fkWh | Relay: %v",
		rm.state.Voltage, rm.state.Current, rm.state.ActivePower, rm.state.PowerFactor,
		rm.state.Frequency, rm.state.TotalEnergy, rm.state.RelayState)
}

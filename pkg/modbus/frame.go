package modbus

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"modbus-energy-meter-sim/pkg/meter"
)

// Modbus Function Codes
const (
	FuncReadHoldingRegisters  byte = 0x03
	FuncWriteSingleRegister   byte = 0x06
	FuncWriteMultipleRegisters byte = 0x10
)

// ProcessRTUFrame takes a raw incoming Modbus RTU byte frame, parses it, routes to the
// matching RegisterManager in DeviceManager, and returns the response frame and PacketLog entries.
func ProcessRTUFrame(rxBytes []byte, dm *meter.DeviceManager, packetCounter *int64) ([]byte, *meter.PacketLog, *meter.PacketLog) {
	if len(rxBytes) < 4 || dm == nil {
		return nil, nil, nil
	}

	slaveID := rxBytes[0]
	rm := dm.GetMeter(slaveID)
	if rm == nil {
		if slaveID == 0 {
			rm = dm.GetDefaultMeter()
		} else {
			// Frame addressed to a different meter not simulated on this bus
			return nil, nil, nil
		}
	}

	return ProcessRTUFrameSingle(rxBytes, rm, packetCounter)
}

// ProcessRTUFrameSingle processes an RTU frame against a specific RegisterManager.
func ProcessRTUFrameSingle(rxBytes []byte, rm *meter.RegisterManager, packetCounter *int64) ([]byte, *meter.PacketLog, *meter.PacketLog) {
	if len(rxBytes) < 4 || rm == nil {
		return nil, nil, nil
	}

	crcValid := CheckCRC(rxBytes)
	slaveID := rxBytes[0]
	funcCode := rxBytes[1]

	state := rm.GetState()
	// If the frame is addressed to another slave ID (and not broadcast 0x00), ignore it
	if slaveID != state.StationAddress && slaveID != 0 {
		return nil, nil, nil
	}

	deviceName := fmt.Sprintf("Meter #%d (%s)", state.StationAddress, state.Name)

	rxLog := &meter.PacketLog{
		Timestamp:   time.Now(),
		Direction:   "RX (STM32 -> Meter)",
		RawHex:      FormatHex(rxBytes),
		Length:      len(rxBytes),
		SlaveID:     slaveID,
		DeviceName:  deviceName,
		Function:    funcCode,
		CRCValid:    crcValid,
		ByteDetails: BreakdownFrame(rxBytes, true),
	}

	if !crcValid {
		rxLog.Description = "CRC Error: invalid checksum received"
		return nil, rxLog, nil
	}

	var txBytes []byte

	switch funcCode {
	case FuncReadHoldingRegisters:
		if len(rxBytes) < 8 {
			txBytes = BuildRTUException(slaveID, funcCode, meter.ExpIllegalDataValue)
			rxLog.Description = "Read Holding Regs (Malformed length)"
		} else {
			startAddr := binary.BigEndian.Uint16(rxBytes[2:4])
			count := binary.BigEndian.Uint16(rxBytes[4:6])
			rxLog.Description = fmt.Sprintf("Read Holding Registers: Start=0x%04X (%s), Count=%d",
				startAddr, RegisterName(startAddr), count)

			data, exp := rm.ReadHoldingRegisters(startAddr, count)
			if exp != 0 {
				txBytes = BuildRTUException(slaveID, funcCode, exp)
			} else {
				respPayload := []byte{slaveID, funcCode, byte(len(data))}
				respPayload = append(respPayload, data...)
				txBytes = AppendCRC(respPayload)
			}
		}

	case FuncWriteMultipleRegisters:
		if len(rxBytes) < 9 {
			txBytes = BuildRTUException(slaveID, funcCode, meter.ExpIllegalDataValue)
			rxLog.Description = "Write Multiple Regs (Malformed length)"
		} else {
			startAddr := binary.BigEndian.Uint16(rxBytes[2:4])
			count := binary.BigEndian.Uint16(rxBytes[4:6])
			byteCount := rxBytes[6]
			expectedLen := 7 + int(byteCount) + 2

			if len(rxBytes) < expectedLen || int(byteCount) != int(count*2) {
				txBytes = BuildRTUException(slaveID, funcCode, meter.ExpIllegalDataValue)
				rxLog.Description = fmt.Sprintf("Write Multiple Regs (Byte count mismatch)")
			} else {
				data := rxBytes[7 : 7+byteCount]
				rxLog.Description = fmt.Sprintf("Write Multiple Regs: Start=0x%04X (%s), Count=%d, Bytes=%d",
					startAddr, RegisterName(startAddr), count, byteCount)

				exp := rm.WriteMultipleRegisters(startAddr, count, data)
				if exp != 0 {
					txBytes = BuildRTUException(slaveID, funcCode, exp)
				} else {
					respPayload := []byte{slaveID, funcCode, rxBytes[2], rxBytes[3], rxBytes[4], rxBytes[5]}
					txBytes = AppendCRC(respPayload)
				}
			}
		}

	case FuncWriteSingleRegister:
		// Per DDS238-2 ZN-S spec: Meter does NOT understand 0x06, only 0x10.
		rxLog.Description = "Write Single Reg (0x06): Rejected per DDS238-2 spec (Illegal Function 0x01)"
		txBytes = BuildRTUException(slaveID, funcCode, meter.ExpIllegalFunction)

	default:
		rxLog.Description = fmt.Sprintf("Unsupported Function Code: 0x%02X", funcCode)
		txBytes = BuildRTUException(slaveID, funcCode, meter.ExpIllegalFunction)
	}

	var txLog *meter.PacketLog
	if len(txBytes) > 0 {
		txLog = &meter.PacketLog{
			Timestamp:   time.Now(),
			Direction:   "TX (Meter -> STM32)",
			RawHex:      FormatHex(txBytes),
			Length:      len(txBytes),
			SlaveID:     txBytes[0],
			DeviceName:  deviceName,
			Function:    txBytes[1],
			CRCValid:    CheckCRC(txBytes),
			Description: DescribeResponse(txBytes),
			ByteDetails: BreakdownFrame(txBytes, false),
		}
	}

	return txBytes, rxLog, txLog
}

// BuildRTUException creates a standard Modbus RTU exception response frame.
func BuildRTUException(slaveID byte, funcCode byte, expCode byte) []byte {
	payload := []byte{slaveID, funcCode | 0x80, expCode}
	return AppendCRC(payload)
}

// FormatHex returns a space-separated uppercase hex representation of a byte array.
func FormatHex(data []byte) string {
	hexes := make([]string, len(data))
	for i, b := range data {
		hexes[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(hexes, " ")
}

// RegisterName returns the description for known DDS238-2 registers.
func RegisterName(addr uint16) string {
	switch addr {
	case meter.RegTotalEnergyHigh:
		return "Total Energy (High)"
	case meter.RegTotalEnergyLow:
		return "Total Energy (Low)"
	case meter.RegExportEnergyHigh:
		return "Export Energy (High)"
	case meter.RegExportEnergyLow:
		return "Export Energy (Low)"
	case meter.RegImportEnergyHigh:
		return "Import Energy (High)"
	case meter.RegImportEnergyLow:
		return "Import Energy (Low)"
	case meter.RegVoltage:
		return "Voltage (0.1 V)"
	case meter.RegCurrent:
		return "Current (0.01 A)"
	case meter.RegActivePower:
		return "Active Power (1 W)"
	case meter.RegReactivePower:
		return "Reactive Power (1 VAr)"
	case meter.RegPowerFactor:
		return "Power Factor (0.001)"
	case meter.RegFrequency:
		return "Frequency (0.01 Hz)"
	case meter.RegAddressBaud:
		return "Station Address & Baud Rate"
	case meter.RegRelay:
		return "Relay Control"
	default:
		return fmt.Sprintf("Register 0x%04X", addr)
	}
}

// DescribeResponse gives a plain-text description of an outgoing Modbus response.
func DescribeResponse(frame []byte) string {
	if len(frame) < 3 {
		return "Incomplete response frame"
	}
	funcCode := frame[1]
	if (funcCode & 0x80) != 0 {
		expCode := frame[2]
		expName := "Unknown"
		switch expCode {
		case meter.ExpIllegalFunction:
			expName = "Illegal Function (0x01)"
		case meter.ExpIllegalDataAddress:
			expName = "Illegal Data Address (0x02)"
		case meter.ExpIllegalDataValue:
			expName = "Illegal Data Value (0x03)"
		case meter.ExpSlaveDeviceFailure:
			expName = "Slave Device Failure (0x04)"
		}
		return fmt.Sprintf("Exception Response: %s for func 0x%02X", expName, funcCode&0x7F)
	}

	switch funcCode {
	case FuncReadHoldingRegisters:
		byteCount := frame[2]
		return fmt.Sprintf("Read Response: %d bytes (%d registers)", byteCount, byteCount/2)
	case FuncWriteMultipleRegisters:
		return "Write Multiple Registers: Success ACK"
	default:
		return fmt.Sprintf("Response for Func 0x%02X", funcCode)
	}
}

// BreakdownFrame analyzes each byte of a Modbus RTU frame for the UI inspector.
func BreakdownFrame(frame []byte, isRequest bool) []meter.ByteDetail {
	details := make([]meter.ByteDetail, len(frame))
	if len(frame) == 0 {
		return details
	}

	for i, b := range frame {
		details[i] = meter.ByteDetail{
			Offset: i,
			Hex:    fmt.Sprintf("%02X", b),
		}
	}

	if len(frame) >= 1 {
		details[0].Label = fmt.Sprintf("Server ID (%d)", frame[0])
	}
	if len(frame) >= 2 {
		fc := frame[1]
		if (fc & 0x80) != 0 {
			details[1].Label = fmt.Sprintf("Exception Flag (Func 0x%02X)", fc&0x7F)
		} else {
			details[1].Label = fmt.Sprintf("Function Code (0x%02X)", fc)
		}
	}

	if isRequest {
		if len(frame) >= 6 && frame[1] == FuncReadHoldingRegisters {
			start := binary.BigEndian.Uint16(frame[2:4])
			count := binary.BigEndian.Uint16(frame[4:6])
			details[2].Label = fmt.Sprintf("Start Addr High (0x%02X)", frame[2])
			details[3].Label = fmt.Sprintf("Start Addr Low -> 0x%04X (%s)", start, RegisterName(start))
			details[4].Label = fmt.Sprintf("Reg Count High (0x%02X)", frame[4])
			details[5].Label = fmt.Sprintf("Reg Count Low -> %d reg(s)", count)
		} else if len(frame) >= 7 && frame[1] == FuncWriteMultipleRegisters {
			start := binary.BigEndian.Uint16(frame[2:4])
			count := binary.BigEndian.Uint16(frame[4:6])
			details[2].Label = fmt.Sprintf("Start Addr High (0x%02X)", frame[2])
			details[3].Label = fmt.Sprintf("Start Addr Low -> 0x%04X", start)
			details[4].Label = fmt.Sprintf("Reg Count High (0x%02X)", frame[4])
			details[5].Label = fmt.Sprintf("Reg Count Low -> %d reg(s)", count)
			details[6].Label = fmt.Sprintf("Byte Count -> %d bytes", frame[6])
			for d := 7; d < len(frame)-2; d++ {
				details[d].Label = fmt.Sprintf("Write Data[%d]", d-7)
			}
		}
	} else {
		if (frame[1] & 0x80) != 0 && len(frame) >= 3 {
			details[2].Label = fmt.Sprintf("Exception Code (0x%02X)", frame[2])
		} else if frame[1] == FuncReadHoldingRegisters && len(frame) >= 3 {
			details[2].Label = fmt.Sprintf("Byte Count (%d bytes)", frame[2])
			for d := 3; d < len(frame)-2; d += 2 {
				regIdx := (d - 3) / 2
				details[d].Label = fmt.Sprintf("Reg[%d] High Byte", regIdx)
				if d+1 < len(frame)-2 {
					val := binary.BigEndian.Uint16(frame[d : d+2])
					details[d+1].Label = fmt.Sprintf("Reg[%d] Low -> 0x%04X (%d)", regIdx, val, val)
				}
			}
		} else if frame[1] == FuncWriteMultipleRegisters && len(frame) >= 6 {
			start := binary.BigEndian.Uint16(frame[2:4])
			count := binary.BigEndian.Uint16(frame[4:6])
			details[2].Label = fmt.Sprintf("Start Addr High (0x%02X)", frame[2])
			details[3].Label = fmt.Sprintf("Start Addr Low -> 0x%04X", start)
			details[4].Label = fmt.Sprintf("Reg Count High (0x%02X)", frame[4])
			details[5].Label = fmt.Sprintf("Reg Count Low -> %d reg(s)", count)
		}
	}

	if len(frame) >= 2 {
		crcLow := frame[len(frame)-2]
		crcHigh := frame[len(frame)-1]
		details[len(frame)-2].Label = fmt.Sprintf("CRC-16 Low (0x%02X)", crcLow)
		details[len(frame)-1].Label = fmt.Sprintf("CRC-16 High (0x%02X)", crcHigh)
	}

	return details
}

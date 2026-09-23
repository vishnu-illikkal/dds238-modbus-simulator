package modbus

import (
	"encoding/binary"
	"fmt"
	"time"

	"modbus-energy-meter-sim/pkg/meter"
)

// CommandResult represents the output of an executed Modbus reader command.
type CommandResult struct {
	Timestamp      time.Time          `json:"timestamp"`
	RequestHex     string             `json:"request_hex"`
	ResponseHex    string             `json:"response_hex"`
	Function       uint8              `json:"function"`
	StartAddr      uint16             `json:"start_addr"`
	RegCount       uint16             `json:"reg_count"`
	Success        bool               `json:"success"`
	ErrorMessage   string             `json:"error_message,omitempty"`
	ParsedValues   map[string]any     `json:"parsed_values,omitempty"`
	RequestDetails []meter.ByteDetail `json:"request_details"`
	ResponseDetails []meter.ByteDetail `json:"response_details"`
	C_Snippet      string             `json:"c_snippet"`
}

// BuildReadHoldingRequest constructs a Modbus RTU Read Holding Registers (0x03) frame with CRC.
func BuildReadHoldingRequest(slaveID uint8, startAddr uint16, count uint16) []byte {
	pdu := []byte{
		slaveID,
		FuncReadHoldingRegisters,
		byte(startAddr >> 8),
		byte(startAddr & 0xFF),
		byte(count >> 8),
		byte(count & 0xFF),
	}
	return AppendCRC(pdu)
}

// BuildWriteMultipleRequest constructs a Modbus RTU Write Multiple Registers (0x10) frame with CRC.
func BuildWriteMultipleRequest(slaveID uint8, startAddr uint16, values []uint16) []byte {
	count := uint16(len(values))
	byteCount := byte(count * 2)

	pdu := []byte{
		slaveID,
		FuncWriteMultipleRegisters,
		byte(startAddr >> 8),
		byte(startAddr & 0xFF),
		byte(count >> 8),
		byte(count & 0xFF),
		byteCount,
	}

	for _, v := range values {
		pdu = append(pdu, byte(v>>8), byte(v&0xFF))
	}

	return AppendCRC(pdu)
}

// ExecuteInternalQuery executes a query directly against the register manager and formats the result.
func ExecuteInternalQuery(rm *meter.RegisterManager, reqBytes []byte) *CommandResult {
	res := &CommandResult{
		Timestamp:      time.Now(),
		RequestHex:     FormatHex(reqBytes),
		RequestDetails: BreakdownFrame(reqBytes, true),
		ParsedValues:   make(map[string]any),
	}

	if len(reqBytes) < 4 {
		res.Success = false
		res.ErrorMessage = "Frame too short"
		return res
	}

	var counter int64
	respBytes, _, _ := ProcessRTUFrameSingle(reqBytes, rm, &counter)

	if len(respBytes) == 0 {
		res.Success = false
		res.ErrorMessage = "No response from meter (Check slave ID or CRC)"
		return res
	}

	res.ResponseHex = FormatHex(respBytes)
	res.ResponseDetails = BreakdownFrame(respBytes, false)
	res.Success = true

	funcCode := reqBytes[1]
	res.Function = funcCode

	// Check if exception
	if (respBytes[1] & 0x80) != 0 {
		res.Success = false
		res.ErrorMessage = DescribeResponse(respBytes)
		return res
	}

	if funcCode == FuncReadHoldingRegisters && len(reqBytes) >= 6 && len(respBytes) >= 5 {
		startAddr := binary.BigEndian.Uint16(reqBytes[2:4])
		count := binary.BigEndian.Uint16(reqBytes[4:6])
		res.StartAddr = startAddr
		res.RegCount = count

		data := respBytes[3 : len(respBytes)-2]
		parseDDS238Values(startAddr, count, data, res.ParsedValues)
	}

	res.C_Snippet = GenerateSTM32Snippet(reqBytes, res)
	return res
}

func parseDDS238Values(startAddr uint16, count uint16, data []byte, out map[string]any) {
	for i := uint16(0); i < count; i++ {
		addr := startAddr + i
		offset := int(i * 2)
		if offset+2 > len(data) {
			break
		}

		switch addr {
		case meter.RegTotalEnergyHigh:
			if offset+4 <= len(data) {
				high := binary.BigEndian.Uint16(data[offset : offset+2])
				low := binary.BigEndian.Uint16(data[offset+2 : offset+4])
				raw := (uint32(high) << 16) | uint32(low)
				kwh := float64(raw) / 100.0
				out["Total Energy"] = fmt.Sprintf("%.2f kWh (raw: %d)", kwh, raw)
			}
		case meter.RegExportEnergyHigh:
			if offset+4 <= len(data) {
				high := binary.BigEndian.Uint16(data[offset : offset+2])
				low := binary.BigEndian.Uint16(data[offset+2 : offset+4])
				raw := (uint32(high) << 16) | uint32(low)
				kwh := float64(raw) / 100.0
				out["Export Energy"] = fmt.Sprintf("%.2f kWh (raw: %d)", kwh, raw)
			}
		case meter.RegImportEnergyHigh:
			if offset+4 <= len(data) {
				high := binary.BigEndian.Uint16(data[offset : offset+2])
				low := binary.BigEndian.Uint16(data[offset+2 : offset+4])
				raw := (uint32(high) << 16) | uint32(low)
				kwh := float64(raw) / 100.0
				out["Import Energy"] = fmt.Sprintf("%.2f kWh (raw: %d)", kwh, raw)
			}
		case meter.RegVoltage:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			out["Voltage"] = fmt.Sprintf("%.1f V (raw: %d)", float64(raw)/10.0, raw)
		case meter.RegCurrent:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			out["Current"] = fmt.Sprintf("%.2f A (raw: %d)", float64(raw)/100.0, raw)
		case meter.RegActivePower:
			raw := int16(binary.BigEndian.Uint16(data[offset : offset+2]))
			out["Active Power"] = fmt.Sprintf("%d W", raw)
		case meter.RegReactivePower:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			out["Reactive Power"] = fmt.Sprintf("%d VAr", raw)
		case meter.RegPowerFactor:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			out["Power Factor"] = fmt.Sprintf("%.3f (raw: %d)", float64(raw)/1000.0, raw)
		case meter.RegFrequency:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			out["Frequency"] = fmt.Sprintf("%.2f Hz (raw: %d)", float64(raw)/100.0, raw)
		case meter.RegAddressBaud:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			station := uint8(raw >> 8)
			baudCode := uint8(raw & 0xFF)
			out["Station / Baud"] = fmt.Sprintf("Addr=%d, Baud=%d Bd", station, meter.BaudRateFromCode(baudCode))
		case meter.RegRelay:
			raw := binary.BigEndian.Uint16(data[offset : offset+2])
			if raw != 0 {
				out["Relay State"] = "ON (Closed)"
			} else {
				out["Relay State"] = "OFF (Open)"
			}
		}
	}
}

// GenerateSTM32Snippet produces copy-paste ready C code for STM32 HAL UART.
func GenerateSTM32Snippet(reqBytes []byte, res *CommandResult) string {
	hexElems := make([]string, len(reqBytes))
	for i, b := range reqBytes {
		hexElems[i] = fmt.Sprintf("0x%02X", b)
	}
	cArray := fmt.Sprintf("uint8_t modbus_req[%d] = { %s };", len(reqBytes), fmt.Sprint(hexElems))

	snippet := fmt.Sprintf(`/* === STM32 HAL Modbus RTU Example === */
// 1. Transmit Frame over UART (9600 Baud, 8N1):
%s
HAL_UART_Transmit(&huart2, modbus_req, sizeof(modbus_req), 100);

// 2. Receive Response:
uint8_t rx_buf[64];
if (HAL_UART_Receive(&huart2, rx_buf, %d, 200) == HAL_OK) {
    // Verify CRC (rx_buf[0..len-3]) matches rx_buf[len-2..len-1]
    // Parse registers starting at rx_buf[3] (Big-Endian)
}
`, cArray, len(reqBytes)+5) // estimate response len

	return snippet
}

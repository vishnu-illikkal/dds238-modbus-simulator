package modbus

// CalculateCRC16 calculates the standard Modbus RTU 16-bit CRC (polynomial 0xA001, init 0xFFFF).
func CalculateCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// AppendCRC appends the 2-byte CRC (Low byte first, High byte second) to the data buffer.
func AppendCRC(data []byte) []byte {
	crc := CalculateCRC16(data)
	low := byte(crc & 0xFF)
	high := byte((crc >> 8) & 0xFF)
	return append(data, low, high)
}

// CheckCRC validates whether the last 2 bytes of the frame match the Modbus RTU CRC.
func CheckCRC(frame []byte) bool {
	if len(frame) < 3 {
		return false
	}
	payload := frame[:len(frame)-2]
	expectedCRC := CalculateCRC16(payload)
	actualCRC := uint16(frame[len(frame)-2]) | (uint16(frame[len(frame)-1]) << 8)
	return expectedCRC == actualCRC
}

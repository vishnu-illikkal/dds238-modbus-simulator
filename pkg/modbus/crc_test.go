package modbus

import (
	"testing"
)

func TestModbusCRC(t *testing.T) {
	// Standard Modbus RTU Read Holding Registers example:
	// Slave: 0x01, Func: 0x03, Start: 0x000C, Count: 0x0001
	req := []byte{0x01, 0x03, 0x00, 0x0C, 0x00, 0x01}
	crc := CalculateCRC16(req)
	// Expected CRC: 0x0944 (Low: 0x44, High: 0x09)
	if crc != 0x0944 {
		t.Fatalf("expected CRC 0x0944, got 0x%04X", crc)
	}

	frame := AppendCRC(req)
	if !CheckCRC(frame) {
		t.Fatalf("CheckCRC failed for valid frame: % X", frame)
	}

	// Corrupt frame
	frame[len(frame)-1] ^= 0xFF
	if CheckCRC(frame) {
		t.Fatalf("CheckCRC should have failed for corrupted frame")
	}
}

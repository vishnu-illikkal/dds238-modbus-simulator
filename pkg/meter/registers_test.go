package meter

import (
	"encoding/binary"
	"testing"
)

func TestRegisterManager_ReadHoldingRegisters(t *testing.T) {
	rm := NewRegisterManager(1)
	rm.UpdateState(func(s *MeterState) {
		s.Voltage = 230.5
		s.Current = 5.20
		s.TotalEnergy = 120.45
	})

	// Test 1: Read Voltage (0x000C, 1 reg) -> 230.5 * 10 = 2305
	data, exp := rm.ReadHoldingRegisters(RegVoltage, 1)
	if exp != 0 {
		t.Fatalf("expected exception 0, got %d", exp)
	}
	vRaw := binary.BigEndian.Uint16(data)
	if vRaw != 2305 {
		t.Fatalf("expected voltage raw 2305, got %d", vRaw)
	}

	// Test 2: Read Total Energy (0x0000, 2 regs) -> 120.45 * 100 = 12045
	data, exp = rm.ReadHoldingRegisters(RegTotalEnergyHigh, 2)
	if exp != 0 {
		t.Fatalf("expected exception 0, got %d", exp)
	}
	high := binary.BigEndian.Uint16(data[0:2])
	low := binary.BigEndian.Uint16(data[2:4])
	total := (uint32(high) << 16) | uint32(low)
	if total != 12045 {
		t.Fatalf("expected total energy raw 12045, got %d", total)
	}

	// Test 3: Out of bound address
	_, exp = rm.ReadHoldingRegisters(0x0020, 1)
	if exp != ExpIllegalDataAddress {
		t.Fatalf("expected ExpIllegalDataAddress, got %d", exp)
	}
}

func TestRegisterManager_WriteMultipleRegisters(t *testing.T) {
	rm := NewRegisterManager(1)

	// Switch relay off (0x001A = 0)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, 0x0000)
	exp := rm.WriteMultipleRegisters(RegRelay, 1, buf)
	if exp != 0 {
		t.Fatalf("expected exception 0, got %d", exp)
	}
	if rm.GetState().RelayState != false {
		t.Fatalf("expected relay state false, got true")
	}

	// Switch relay back on (0x001A = 1)
	binary.BigEndian.PutUint16(buf, 0x0001)
	exp = rm.WriteMultipleRegisters(RegRelay, 1, buf)
	if exp != 0 {
		t.Fatalf("expected exception 0, got %d", exp)
	}
	if rm.GetState().RelayState != true {
		t.Fatalf("expected relay state true, got false")
	}

	// Clear energy counters by writing 0 to 0x0000 (2 registers)
	zeroBuf := make([]byte, 4)
	exp = rm.WriteMultipleRegisters(RegTotalEnergyHigh, 2, zeroBuf)
	if exp != 0 {
		t.Fatalf("expected exception 0, got %d", exp)
	}
	state := rm.GetState()
	if state.TotalEnergy != 0 || state.ExportEnergy != 0 || state.ImportEnergy != 0 {
		t.Fatalf("expected energy counters to be 0, got Total=%.2f, Export=%.2f, Import=%.2f",
			state.TotalEnergy, state.ExportEnergy, state.ImportEnergy)
	}
}

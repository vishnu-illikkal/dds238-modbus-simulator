package modbus

import (
	"encoding/binary"
	"testing"

	"modbus-energy-meter-sim/pkg/meter"
)

func TestProcessRTUFrame_ReadVoltage(t *testing.T) {
	rm := meter.NewRegisterManager(1)
	rm.UpdateState(func(s *meter.MeterState) {
		s.Voltage = 230.0 // 2300 = 0x08FC
	})

	// Request: Read 1 register at 0x000C
	// Slave: 0x01, Func: 0x03, Start: 0x000C, Count: 0x0001
	req := BuildReadHoldingRequest(1, meter.RegVoltage, 1)

	var counter int64
	resp, rxLog, txLog := ProcessRTUFrameSingle(req, rm, &counter)

	if rxLog == nil || txLog == nil {
		t.Fatalf("expected packet logs, got nil")
	}

	if len(resp) != 7 {
		t.Fatalf("expected 7 byte response, got %d bytes: % X", len(resp), resp)
	}

	if !CheckCRC(resp) {
		t.Fatalf("response failed CRC check: % X", resp)
	}

	// Response: [01][03][02][08 FC][CRC Low][CRC High]
	if resp[0] != 0x01 || resp[1] != 0x03 || resp[2] != 0x02 {
		t.Fatalf("unexpected header in response: % X", resp)
	}

	vVal := binary.BigEndian.Uint16(resp[3:5])
	if vVal != 2300 {
		t.Fatalf("expected voltage 2300, got %d", vVal)
	}
}

func TestProcessRTUFrame_WriteSingleRegister_Rejection(t *testing.T) {
	rm := meter.NewRegisterManager(1)

	// Function 0x06 (Write Single Register) -> DDS238-2 rejects this
	req := []byte{0x01, 0x06, 0x00, 0x1A, 0x00, 0x01}
	req = AppendCRC(req)

	var counter int64
	resp, rxLog, txLog := ProcessRTUFrameSingle(req, rm, &counter)

	if rxLog == nil || txLog == nil {
		t.Fatalf("expected packet logs, got nil")
	}

	// Exception response: [01][86][01 (Illegal Function)][CRC Low][CRC High]
	if len(resp) != 5 {
		t.Fatalf("expected 5 byte exception response, got %d bytes: % X", len(resp), resp)
	}

	if resp[1] != 0x86 || resp[2] != meter.ExpIllegalFunction {
		t.Fatalf("expected 0x86 0x01 exception, got % X", resp)
	}
}

func TestProcessRTUFrame_WriteMultipleRegisters_Relay(t *testing.T) {
	rm := meter.NewRegisterManager(1)

	// Write 1 register at 0x001A with value 0 (Turn relay off)
	req := BuildWriteMultipleRequest(1, meter.RegRelay, []uint16{0x0000})

	var counter int64
	resp, _, _ := ProcessRTUFrameSingle(req, rm, &counter)

	if len(resp) != 8 {
		t.Fatalf("expected 8 byte write multiple response, got %d bytes: % X", len(resp), resp)
	}

	if !CheckCRC(resp) {
		t.Fatalf("response failed CRC check: % X", resp)
	}

	if rm.GetState().RelayState != false {
		t.Fatalf("expected relay state to be false")
	}
}

func TestProcessRTUFrame_MultiMeterRouting(t *testing.T) {
	dm := meter.NewDeviceManager(1, 2)
	m1 := dm.GetMeter(1)
	m2 := dm.GetMeter(2)

	m1.UpdateState(func(s *meter.MeterState) {
		s.Voltage = 230.0
	})
	m2.UpdateState(func(s *meter.MeterState) {
		s.Voltage = 240.0
	})

	var counter int64

	// Query Meter 1
	req1 := BuildReadHoldingRequest(1, meter.RegVoltage, 1)
	resp1, rx1, tx1 := ProcessRTUFrame(req1, dm, &counter)
	if len(resp1) != 7 || rx1 == nil || tx1 == nil {
		t.Fatalf("failed querying Meter 1")
	}
	if binary.BigEndian.Uint16(resp1[3:5]) != 2300 {
		t.Fatalf("expected Meter 1 voltage 2300, got %d", binary.BigEndian.Uint16(resp1[3:5]))
	}

	// Query Meter 2
	req2 := BuildReadHoldingRequest(2, meter.RegVoltage, 1)
	resp2, rx2, tx2 := ProcessRTUFrame(req2, dm, &counter)
	if len(resp2) != 7 || rx2 == nil || tx2 == nil {
		t.Fatalf("failed querying Meter 2")
	}
	if binary.BigEndian.Uint16(resp2[3:5]) != 2400 {
		t.Fatalf("expected Meter 2 voltage 2400, got %d", binary.BigEndian.Uint16(resp2[3:5]))
	}

	// Query non-existent Meter 3 -> Should return nil (no answer on bus)
	req3 := BuildReadHoldingRequest(3, meter.RegVoltage, 1)
	resp3, rx3, tx3 := ProcessRTUFrame(req3, dm, &counter)
	if resp3 != nil || rx3 != nil || tx3 != nil {
		t.Fatalf("expected nil response for non-existent meter 3, got: % X", resp3)
	}
}

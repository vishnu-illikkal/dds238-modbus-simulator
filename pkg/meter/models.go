package meter

import "time"

// MeterState holds the physical measurements and configuration of the simulated DDS238-2 meter.
type MeterState struct {
	Name         string  `json:"name"`              // Meter alias / role (e.g., "Main Grid", "Solar PV")
	TotalEnergy  float64 `json:"total_energy_kwh"`  // kWh
	ExportEnergy float64 `json:"export_energy_kwh"` // kWh
	ImportEnergy float64 `json:"import_energy_kwh"` // kWh

	Voltage       float64 `json:"voltage_v"`        // V
	Current       float64 `json:"current_a"`        // A
	ActivePower   float64 `json:"active_power_w"`   // W (signed)
	ReactivePower float64 `json:"reactive_power_var"`// VAr
	PowerFactor   float64 `json:"power_factor"`     // 0.000 to 1.000
	Frequency     float64 `json:"frequency_hz"`     // Hz

	StationAddress uint8 `json:"station_address"` // 1-247 (default 1)
	BaudRateCode   uint8 `json:"baud_rate_code"`  // 1: 9600, 2: 4800, 3: 2400, 4: 1200
	RelayState     bool  `json:"relay_state"`     // false = Off (0), true = On (1)

	// Simulation controls
	SimulateNoise     bool    `json:"simulate_noise"`     // Add subtle grid fluctuations
	DynamicEnergy     bool    `json:"dynamic_energy"`     // Automatically accumulate energy over time
	TargetCurrent     float64 `json:"target_current_a"`   // Setpoint current
	TargetVoltage     float64 `json:"target_voltage_v"`   // Setpoint voltage
	TargetPowerFactor float64 `json:"target_pf"`          // Setpoint PF
	TargetFrequency   float64 `json:"target_frequency_hz"`// Setpoint Frequency

	LastUpdated time.Time `json:"last_updated"`
}

// PacketLog represents a logged Modbus RTU communication packet.
type PacketLog struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Direction   string    `json:"direction"` // "RX" (from STM32) or "TX" (to STM32)
	RawHex      string    `json:"raw_hex"`
	Length      int       `json:"length"`
	SlaveID     uint8     `json:"slave_id"`
	DeviceName  string    `json:"device_name"` // e.g. "Meter #1 (Main Grid)"
	Function    uint8     `json:"function"`
	Description string    `json:"description"`
	CRCValid    bool      `json:"crc_valid"`
	ByteDetails []ByteDetail `json:"byte_details"`
}

// ByteDetail provides human-readable context for each byte in a Modbus frame.
type ByteDetail struct {
	Offset int    `json:"offset"`
	Hex    string `json:"hex"`
	Label  string `json:"label"`
}

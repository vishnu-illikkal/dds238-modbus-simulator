package meter

import (
	"fmt"
	"sort"
	"sync"
)

// DeviceManager manages multiple simulated DDS238 energy meters on the same bus.
type DeviceManager struct {
	mu     sync.RWMutex
	meters map[uint8]*RegisterManager
}

// NewDeviceManager initializes the device manager with one or more starting meter IDs.
func NewDeviceManager(initialIDs ...uint8) *DeviceManager {
	dm := &DeviceManager{
		meters: make(map[uint8]*RegisterManager),
	}

	if len(initialIDs) == 0 {
		initialIDs = []uint8{1}
	}

	defaultNames := map[uint8]string{
		1: "Main Grid",
		2: "Solar PV Inverter",
		3: "EV Fast Charger",
		4: "Heat Pump & HVAC",
		5: "Battery Storage",
	}

	for _, id := range initialIDs {
		if id == 0 {
			id = 1
		}
		name, exists := defaultNames[id]
		if !exists {
			name = fmt.Sprintf("Meter #%d", id)
		}
		dm.meters[id] = NewNamedRegisterManager(id, name)
	}

	return dm
}

// GetMeter returns the RegisterManager for a specific Server/Slave ID.
func (dm *DeviceManager) GetMeter(id uint8) *RegisterManager {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.meters[id]
}

// GetDefaultMeter returns the first available meter, or creates Meter #1 if none exist.
func (dm *DeviceManager) GetDefaultMeter() *RegisterManager {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if len(dm.meters) == 0 {
		m := NewNamedRegisterManager(1, "Main Grid")
		dm.meters[1] = m
		return m
	}

	// Return meter with lowest ID
	var lowestID uint8 = 255
	for id := range dm.meters {
		if id < lowestID {
			lowestID = id
		}
	}
	return dm.meters[lowestID]
}

// GetAllMeters returns all active RegisterManagers sorted by Server ID.
func (dm *DeviceManager) GetAllMeters() []*RegisterManager {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	keys := make([]int, 0, len(dm.meters))
	for k := range dm.meters {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	list := make([]*RegisterManager, 0, len(keys))
	for _, k := range keys {
		list = append(list, dm.meters[uint8(k)])
	}
	return list
}

// GetAllStates returns state snapshots of all registered meters.
func (dm *DeviceManager) GetAllStates() []MeterState {
	meters := dm.GetAllMeters()
	states := make([]MeterState, len(meters))
	for i, m := range meters {
		states[i] = m.GetState()
	}
	return states
}

// AddMeter creates and adds a new meter instance with the given Server ID.
func (dm *DeviceManager) AddMeter(id uint8, name string) (*RegisterManager, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if id < 1 || id > 247 {
		return nil, fmt.Errorf("invalid Server ID %d (must be 1-247)", id)
	}

	if _, exists := dm.meters[id]; exists {
		return nil, fmt.Errorf("meter with Server ID %d already exists", id)
	}

	if name == "" {
		name = fmt.Sprintf("Meter #%d", id)
	}

	m := NewNamedRegisterManager(id, name)
	
	// If Solar PV role is selected or preset, initialize with realistic solar export telemetry
	if name == "Solar PV Inverter" || name == "Solar PV" {
		m.UpdateState(func(s *MeterState) {
			s.TargetVoltage = 232.0
			s.TargetCurrent = 15.0
			s.Voltage = 232.0
			s.Current = 15.0
			s.ActivePower = -3480 // Negative active power = Exporting to grid
			s.ExportEnergy = 450.25
			s.ImportEnergy = 12.10
			s.TotalEnergy = 462.35
		})
	}

	dm.meters[id] = m
	return m, nil
}

// RemoveMeter removes a meter by Server ID.
func (dm *DeviceManager) RemoveMeter(id uint8) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if len(dm.meters) <= 1 {
		return fmt.Errorf("cannot remove the last active meter")
	}

	if _, exists := dm.meters[id]; !exists {
		return fmt.Errorf("meter with Server ID %d not found", id)
	}

	delete(dm.meters, id)
	return nil
}

// Count returns the number of active meters.
func (dm *DeviceManager) Count() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return len(dm.meters)
}

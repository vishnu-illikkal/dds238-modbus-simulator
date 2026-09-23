package meter

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// SimulationEngine periodically updates meter measurements to simulate a live AC grid & load.
type SimulationEngine struct {
	dm     *DeviceManager
	rng    *rand.Rand
	ticker *time.Ticker
}

// NewSimulationEngine initializes the simulation engine.
func NewSimulationEngine(dm *DeviceManager) *SimulationEngine {
	return &SimulationEngine{
		dm:  dm,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Start runs the simulation loop until ctx is cancelled.
func (se *SimulationEngine) Start(ctx context.Context, interval time.Duration) {
	se.ticker = time.NewTicker(interval)
	go func() {
		lastTime := time.Now()
		for {
			select {
			case <-ctx.Done():
				se.ticker.Stop()
				return
			case now := <-se.ticker.C:
				dtHours := now.Sub(lastTime).Hours()
				lastTime = now
				se.stepAll(dtHours)
			}
		}
	}()
}

func (se *SimulationEngine) stepAll(dtHours float64) {
	for _, rm := range se.dm.GetAllMeters() {
		se.stepMeter(rm, dtHours)
	}
}

func (se *SimulationEngine) stepMeter(rm *RegisterManager, dtHours float64) {
	rm.UpdateState(func(s *MeterState) {
		if !s.RelayState {
			// Relay is open -> Zero current and power
			s.Current = 0
			s.ActivePower = 0
			s.ReactivePower = 0
			return
		}

		// Calculate realistic grid voltage with subtle Gaussian jitter if noise is enabled
		vBase := s.TargetVoltage
		if s.SimulateNoise {
			vNoise := (se.rng.NormFloat64() * 0.35)
			s.Voltage = math.Round((vBase+vNoise)*10) / 10
		} else {
			s.Voltage = vBase
		}

		// Calculate Current with noise
		iBase := s.TargetCurrent
		if s.SimulateNoise && iBase > 0.05 {
			iNoise := (se.rng.NormFloat64() * 0.02)
			s.Current = math.Max(0, math.Round((iBase+iNoise)*100)/100)
		} else {
			s.Current = iBase
		}

		// Calculate Power Factor with slight variation
		pfBase := s.TargetPowerFactor
		if s.SimulateNoise && iBase > 0.05 {
			pfNoise := (se.rng.NormFloat64() * 0.002)
			s.PowerFactor = math.Max(0.1, math.Min(1.0, pfBase+pfNoise))
		} else {
			s.PowerFactor = pfBase
		}

		// Frequency (50Hz / 60Hz nominal with minor grid frequency drift)
		fBase := s.TargetFrequency
		if s.SimulateNoise {
			fNoise := (se.rng.NormFloat64() * 0.015)
			s.Frequency = math.Round((fBase+fNoise)*100) / 100
		} else {
			s.Frequency = fBase
		}

		// Apparent Power S = V * I
		apparentPower := s.Voltage * s.Current
		
		// If Solar PV role, calculate active power as negative (export to grid)
		if s.Name == "Solar PV Inverter" || s.Name == "Solar PV" {
			s.ActivePower = -math.Round(apparentPower * s.PowerFactor)
		} else {
			// Active Power P = V * I * PF (Watts)
			s.ActivePower = math.Round(apparentPower * s.PowerFactor)
		}

		// Reactive Power Q = sqrt(S^2 - P^2) (VAr)
		absP := math.Abs(s.ActivePower)
		if apparentPower > absP {
			s.ReactivePower = math.Round(math.Sqrt(apparentPower*apparentPower - absP*absP))
		} else {
			s.ReactivePower = 0
		}

		// Dynamic Energy Accumulation: E = P * dt (kWh)
		if s.DynamicEnergy && dtHours > 0 {
			if s.ActivePower > 0 {
				// Import from grid
				kwhDelta := (s.ActivePower / 1000.0) * dtHours
				s.ImportEnergy += kwhDelta
				s.TotalEnergy += kwhDelta
			} else if s.ActivePower < 0 {
				// Export to grid (solar)
				kwhDelta := (math.Abs(s.ActivePower) / 1000.0) * dtHours
				s.ExportEnergy += kwhDelta
				s.TotalEnergy += kwhDelta
			}
		}
	})
}

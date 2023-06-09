package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateKeba(base *Keba, meter func() (float64, error), meterEnergy func() (float64, error), phaseCurrents func() (float64, float64, float64, error)) api.Charger {
	switch {
	case meter == nil && meterEnergy == nil && phaseCurrents == nil:
		return base

	case meter != nil && meterEnergy == nil && phaseCurrents == nil:
		return &struct {
			*Keba
			api.Meter
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents == nil:
		return &struct {
			*Keba
			api.MeterEnergy
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents == nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents != nil:
		return &struct {
			*Keba
			api.PhaseCurrents
		}{
			Keba: base,
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents != nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseCurrents
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents != nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseCurrents
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents != nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseCurrents
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}
	}

	return nil
}

type decorateKebaMeterImpl struct {
	meter func() (float64, error)
}

func (impl *decorateKebaMeterImpl) CurrentPower() (float64, error) {
	return impl.meter()
}

type decorateKebaMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateKebaMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}

type decorateKebaPhaseCurrentsImpl struct {
	phaseCurrents func() (float64, float64, float64, error)
}

func (impl *decorateKebaPhaseCurrentsImpl) Currents() (float64, float64, float64, error) {
	return impl.phaseCurrents()
}

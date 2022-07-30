package core

import (
	"fmt"
	"math"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// interface to get the current in use from a consumer
// it is expected to get the max current over all phases
type Consumer interface {
	GetCurrent() float64
}

type Circuit struct {
	Log    *util.Logger
	uiChan chan<- util.Param

	Name       string    `mapstructure:"name"`       // meaningful name
	MaxCurrent float64   `mapstructure:"maxCurrent"` // the max allowed current of this circuit
	MeterRef   string    `mapstructure:"meter"`      // Charge meter reference
	Circuits   []Circuit `mapstructure:"circuits"`   // sub circuits as config reference

	parentCircuit *Circuit         // parent circuit reference
	meterCurrent  api.MeterCurrent // optional: meter to determine phase current
	Consumers     []Consumer       // optional: all consumers under management. Used for consumption evaluation if no meter is present
}

// determines current in use. Its defined as consumption by consumers
// if there is a meter, the meter current will define the consumption
// TBD: phase perfect evaluation
func (cc *Circuit) GetCurrent() float64 {
	var current float64
	if cc.meterCurrent != nil {
		current, _ = cc.GetCurrentFromMeter()
	} else {
		current = cc.GetCurrentFromConsumers()
	}
	cc.Log.TRACE.Printf("actual current: %.1fA", current)
	cc.publish("actualCurrent", current)
	return current
}

// returns aggregated consumption of all consumers
// using their current
func (cc *Circuit) GetCurrentFromConsumers() float64 {
	cc.Log.TRACE.Printf("get current from %d consumers", len(cc.Consumers))
	var current float64
	for _, curLp := range cc.Consumers {
		cur := curLp.GetCurrent()
		cc.Log.TRACE.Printf("add %.1fA current from consumer", cur)
		current += cur
	}
	// also add consumption of circuits
	for ccId, _ := range cc.Circuits {
		cur := cc.Circuits[ccId].GetCurrent()
		cc.Log.TRACE.Printf("add %.1fA current from circuit %s", cur, cc.Circuits[ccId].Name)
		current += cur
	}
	cc.publish("consumerCurrent", current)
	return current
}

// returns the highest current from meter
// err if no meter set
func (cc *Circuit) GetCurrentFromMeter() (float64, error) {
	if cc.meterCurrent == nil {
		return 0.0, fmt.Errorf("no meter available")
	}
	cc.Log.TRACE.Printf("get current from meter")
	current := 0.0
	var err error
	i1, i2, i3, err := cc.meterCurrent.Currents()
	if err == nil {
		cc.Log.DEBUG.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
		cc.publish("meterCurrents", []float64{i1, i2, i3})
		// TODO: phase adjusted handling. Currently we take highest current from all phases
		current = math.Max(math.Max(i1, i2), i3)
	} else {
		cc.Log.ERROR.Println(fmt.Errorf("grid meter currents: %v", err))
		return 0.0, fmt.Errorf("error getting meter currents: %v", err)
	}
	return current, nil
}

// returns avaialble current based on limit and consumption
// todo: GetRemainingCurrent
func (cc *Circuit) GetRemainingCurrent() float64 {
	cc.Log.TRACE.Printf("get available current")
	// first update current current, mainly to regularly publish the value
	current := cc.GetCurrent()
	curAvailable := cc.MaxCurrent - current
	if curAvailable < 0.0 {
		cc.Log.WARN.Printf("overload detected (%s) - currents: %.1fA, allowed max current is: %.1fA\n", cc.Name, current, cc.MaxCurrent)
		cc.publish("overload", true)
	} else {
		cc.publish("overload", false)
	}
	// check parent circuit, return lowest
	if cc.parentCircuit != nil {
		cc.Log.TRACE.Printf("get avialable current from parent: %s", cc.parentCircuit.Name)
		curAvailable = math.Min(curAvailable, cc.parentCircuit.GetRemainingCurrent())
	}
	cc.Log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
	return curAvailable
}

// creates a circuit with defaults
func NewCircuit(n string, limit float64, mc api.MeterCurrent, l *util.Logger) *Circuit {
	cc := &Circuit{
		Name:         n,
		Log:          l,
		MaxCurrent:   limit,
		meterCurrent: mc,
	}
	return cc
}

// creates circuit from config
// using site to get access to the grid meter if configured, see cp.Meter() for details
func NewCircuitFromConfig(cp configProvider, other map[string]interface{}, site *Site) (*Circuit, error) {
	cc := NewCircuit("", 0, nil, nil)
	if err := util.DecodeOther(other, cc); err != nil {
		return nil, err
	}

	cc.Log = util.NewLogger("cc-" + cc.Name)
	cc.Log.TRACE.Println("NewCircuitFromConfig()")
	cc.PrintCircuits(0)
	if err := cc.InitCircuits(site, cp); err != nil {
		return nil, err
	}
	cc.PrintCircuits(0)
	cc.Log.TRACE.Printf("created new circuit: %s, limit: %.1fA", cc.Name, cc.MaxCurrent)
	if cc.Log == nil {
		fmt.Println("log is nil")
	}
	cc.Log.TRACE.Println("NewCircuitFromConfig()) end")
	return cc, nil
}

// circuits are recursive, so initialize meters also recursive
func (cc *Circuit) InitCircuits(site *Site, cp configProvider) error {
	if cc.Name == "" {
		return fmt.Errorf("circuit name must not be empty")
	}
	cc.Log = util.NewLogger("cc-" + cc.Name)
	cc.Log.TRACE.Printf("InitCircuits(): %s (%p)", cc.Name, cc)
	if cc.MeterRef != "" {
		var mtr api.Meter
		if cc.MeterRef == site.Meters.GridMeterRef {
			mtr = site.gridMeter
			cc.Log.TRACE.Printf("add grid meter from site: %s", cc.MeterRef)
		} else {
			mtr, _ = cp.Meter(cc.MeterRef)
			cc.Log.TRACE.Printf("add separate meter: %s", cc.MeterRef)
		}
		if mc, ok := mtr.(api.MeterCurrent); ok {
			cc.meterCurrent = mc
		} else {
			return fmt.Errorf("circuit needs meter with phase current support: %s", cc.MeterRef)
		}
	} else {
		cc.Log.TRACE.Printf("no meter specified")
	}
	// initialize also included circuits
	if cc.Circuits != nil {
		for ccId, _ := range cc.Circuits {
			cc.Log.TRACE.Printf("creating circuit from circuitRef: %s", cc.Circuits[ccId].Name)
			cc.Circuits[ccId].parentCircuit = cc
			if err := cc.Circuits[ccId].InitCircuits(site, cp); err != nil {
				return err
			}
			cc.Circuits[ccId].PrintCircuits(0)
		}
	} else {
		cc.Log.TRACE.Printf("no sub circuits")
	}
	cc.Log.TRACE.Println("InitCircuits() exit")
	cc.Log.INFO.Printf("initialized new circuit: %s, limit: %.1fA", cc.Name, cc.MaxCurrent)
	return nil
}

// trace output of circuit and subcircuits
func (cc *Circuit) PrintCircuits(indent int) {
	for _, s := range cc.DumpConfig(0, 15) {
		cc.Log.TRACE.Println(s)
	}
}

// returns string array to dump the config
func (cc *Circuit) DumpConfig(indent int, maxIndent int) []string {

	var res []string

	cfgDump := fmt.Sprintf("%s%s:%s meter %s maxCurrent %.1fA consumers %d",
		strings.Repeat(" ", indent),
		cc.Name,
		strings.Repeat(" ", maxIndent-len(cc.Name)-indent),
		presence[cc.meterCurrent != nil],
		cc.MaxCurrent,
		len(cc.Consumers))
	res = append(res, cfgDump)

	// cc.Log.TRACE.Printf("%s%s%s: (%p) log: %t, meter: %t, consumers: %d, parent: %p\n", strings.Repeat(" ", indent), cc.Name, strings.Repeat(" ", 10-indent), cc, cc.Log != nil, cc.meterCurrent != nil, len(cc.Consumers), cc.parentCircuit)
	for id, _ := range cc.Circuits {
		for _, s := range cc.Circuits[id].DumpConfig(indent+2, maxIndent) {
			res = append(res, s)
		}
	}
	return res
}

// returns the circiut with given name, checking all subcircuits
func (cc *Circuit) GetCircuit(n string) *Circuit {
	cc.Log.TRACE.Printf("searching for circuit %s in %s", n, cc.Name)
	if cc.Name == n {
		cc.Log.TRACE.Printf("found circuit %s (%p)", cc.Name, &cc)
		return cc
	} else {
		for ccId, _ := range cc.Circuits {
			cc.Log.TRACE.Printf("start looking in circuit %s (%p)", cc.Circuits[ccId].Name, &cc.Circuits[ccId])
			retCC := cc.Circuits[ccId].GetCircuit(n)
			if retCC != nil {
				cc.Log.TRACE.Printf("found circuit %s (%p)", retCC.Name, &retCC)
				return retCC
			}
		}
	}
	cc.Log.INFO.Printf("could not find circuit %s", n)
	return nil
}

// publish sends values to UI and databases
func (cc *Circuit) publish(key string, val interface{}) {
	// test helper
	if cc.uiChan == nil {
		return
	}

	key = fmt.Sprintf("circuit-%s_%s", cc.Name, key)

	cc.uiChan <- util.Param{
		Key: key,
		Val: val,
	}
}

// set the UI channel to publish information
func (cc *Circuit) Prepare(uiChan chan<- util.Param) {
	cc.uiChan = uiChan
	cc.publish("maxCurrent", cc.MaxCurrent)
}
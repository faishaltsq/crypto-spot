package outcome

import (
	"testing"

	"github.com/example/crypto-spot-signal/internal/execution_simulation"
)

func TestApplyNetReturnLeavesIncompleteSimulationUnset(t *testing.T) {
	returns := map[Horizon]HorizonReturn{Horizon1m: {Horizon: Horizon1m}}
	applyNetReturn(returns, []execution_simulation.Simulation{{Status: execution_simulation.StatusIncomplete}})
	if returns[Horizon1m].NetReturnPct != nil { t.Fatal("incomplete simulation must not become zero return") }
}

func TestApplyNetReturnFormatsDecimalAsPercentage(t *testing.T) {
	net := 0.0125
	returns := map[Horizon]HorizonReturn{Horizon1m: {Horizon: Horizon1m}}
	applyNetReturn(returns, []execution_simulation.Simulation{{Status: execution_simulation.StatusComplete, GrossReturn: &net, NetReturn: &net}})
	if returns[Horizon1m].NetReturnPct == nil || *returns[Horizon1m].NetReturnPct != 1.25 { t.Fatalf("expected 1.25 percent, got %#v", returns[Horizon1m].NetReturnPct) }
}

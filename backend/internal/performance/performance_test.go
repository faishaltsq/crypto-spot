package performance

import (
	"math"
	"testing"
	"time"
)

func number(v float64) *float64 { return &v }

func TestBuildUsesNetReturnAndExcludesIncomplete(t *testing.T) {
	report := Build([]Sample{
		{SimulationStatus: Evaluated, GrossReturn: number(.03), NetReturn: number(.01), EntryFee: .5, ExitFee: .5, EntrySlippage: .25, ExitSlippage: .25},
		{SimulationStatus: IncompleteSimulation, GrossReturn: number(.90), NetReturn: number(.90)},
		{SimulationStatus: PendingSimulation},
	})
	if got := report.Metrics[0].Value; got != 1 {
		t.Fatalf("evaluated signals = %v, want 1", got)
	}
	for _, metric := range report.Metrics {
		if metric.Name == "average_net_return" && math.Abs(metric.Value-.01) > 1e-12 {
			t.Fatalf("net return = %v, want .01", metric.Value)
		}
		if metric.Name == "total_transaction_cost" && math.Abs(metric.Value-1.5) > 1e-12 {
			t.Fatalf("cost = %v, want 1.5", metric.Value)
		}
	}
	if report.StatusCounts[IncompleteSimulation] != 1 || report.StatusCounts[PendingSimulation] != 1 {
		t.Fatalf("status counts = %#v", report.StatusCounts)
	}
}

func TestPerformanceMath(t *testing.T) {
	if ScoreBucket(69) != "OUT_OF_RANGE" || ScoreBucket(84) != "80-84" || ScoreBucket(90) != "90-100" {
		t.Fatal("score bucket")
	}
	if Reliability(29) != "INSUFFICIENT" || Reliability(500) != "STRONGER_EVIDENCE" {
		t.Fatal("reliability")
	}
	if got := median([]float64{.03, -.01, .01}); got != .01 {
		t.Fatalf("median = %v", got)
	}
	if got := drawdown([]float64{.02, .01, .04}); got[1] != -.01 {
		t.Fatalf("drawdown = %#v", got)
	}
}

func TestBuildIncludesHorizonRowsAndDuration(t *testing.T) {
	report := Build([]Sample{{SimulationStatus: Evaluated, GrossReturn: number(.01), NetReturn: number(.009), Duration: 5 * time.Minute, HorizonReturns: map[string]HorizonSample{"5m": HorizonSample{GrossReturn: .012, NetReturn: .01}}}})
	if len(report.Horizons) != 8 {
		t.Fatalf("horizons = %d", len(report.Horizons))
	}
	if report.Horizons[1].SampleCount != 1 || report.Horizons[1].MeanNetReturn != .01 {
		t.Fatalf("5m horizon = %#v", report.Horizons[1])
	}
	for _, metric := range report.Metrics {
		if metric.Name == "average_signal_duration" && metric.Value != 300 {
			t.Fatalf("duration = %v", metric.Value)
		}
	}
}

func TestConfidenceIntervalRequiresTwoSamples(t *testing.T) {
	one := confidenceInterval([]float64{.01})
	if one != nil {
		t.Fatalf("single sample CI = %#v", one)
	}
	two := confidenceInterval([]float64{.01, .03})
	if two == nil || two[0] >= two[1] {
		t.Fatalf("two sample CI = %#v", two)
	}
}

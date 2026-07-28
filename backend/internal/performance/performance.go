// Package performance calculates proof-of-edge metrics from execution simulations.
package performance

import (
	"math"
	"sort"
	"strconv"
	"time"
)

type Status string

const (
	PendingSimulation    Status = "PENDING_SIMULATION"
	IncompleteSimulation Status = "INCOMPLETE_SIMULATION"
	PartialFill          Status = "PARTIAL_FILL"
	Evaluated            Status = "EVALUATED"
)

type Filters struct {
	DateFrom, DateTo                                                                                                *time.Time `json:"-"`
	Pair, Tier, Timeframe, SignalStatus, ScoreBucket, MarketRegime, RuleVersion, ModelVersion, AIDecision, Notional string     `json:"-"`
}

type Sample struct {
	CreatedAt                                                                                                                    time.Time
	Pair, Tier, Timeframe, SignalStatus, MarketRegime, RuleVersion, ModelVersion, AIDecision, AIProvider, DataQuality, SpoofRisk string
	Score, Notional                                                                                                              float64
	SimulationStatus                                                                                                             Status
	GrossReturn, NetReturn                                                                                                       *float64
	HorizonReturns                                                                                                               map[string]HorizonSample
	EntryFee, ExitFee, EntrySlippage, ExitSlippage                                                                               float64
	MFE, MAE                                                                                                                     float64
	TargetHit, InvalidationHit                                                                                                   bool
	Duration                                                                                                                     time.Duration
}

type HorizonSample struct {
	GrossReturn float64
	NetReturn   float64
}

type Metric struct {
	Name        string  `json:"name"`
	Definition  string  `json:"definition"`
	Unit        string  `json:"unit"`
	Value       float64 `json:"value"`
	SampleCount int     `json:"sampleCount"`
}

type Breakdown struct {
	Dimension                                     string  `json:"dimension"`
	Value                                         string  `json:"value"`
	SampleCount                                   int     `json:"sampleCount"`
	AverageGrossReturn, AverageNetReturn, WinRate float64 `json:"averageGrossReturn"`
}

type Horizon struct {
	Horizon            string      `json:"horizon"`
	MeanGrossReturn    float64     `json:"meanGrossReturn"`
	MedianGrossReturn  float64     `json:"medianGrossReturn"`
	MeanNetReturn      float64     `json:"meanNetReturn"`
	MedianNetReturn    float64     `json:"medianNetReturn"`
	PositiveRate       float64     `json:"positiveRate"`
	SampleCount        int         `json:"sampleCount"`
	ConfidenceInterval *[2]float64 `json:"confidenceInterval,omitempty"`
}

type EdgeComponent struct {
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Score        float64 `json:"score"`
	Contribution float64 `json:"contribution"`
}
type EdgeScore struct {
	Score      float64         `json:"score"`
	Components []EdgeComponent `json:"components"`
}

type Report struct {
	Metrics               []Metric       `json:"metrics"`
	StatusCounts          map[Status]int `json:"statusCounts"`
	Reliability           string         `json:"reliabilityStatus"`
	ReliabilityDefinition string         `json:"reliabilityDefinition"`
	EdgeScore             EdgeScore      `json:"edgeScore"`
	Breakdowns            []Breakdown    `json:"breakdowns"`
	Horizons              []Horizon      `json:"horizons"`
	Warnings              []string       `json:"warnings"`
	CumulativeNet         []float64      `json:"cumulativeNetReturn"`
	CumulativeGross       []float64      `json:"cumulativeGrossReturn"`
	Drawdown              []float64      `json:"drawdown"`
	CalculationTimestamp  time.Time      `json:"calculationTimestamp"`
}

func ScoreBucket(score float64) string {
	switch {
	case score >= 70 && score < 75:
		return "70-74"
	case score >= 75 && score < 80:
		return "75-79"
	case score >= 80 && score < 85:
		return "80-84"
	case score >= 85 && score < 90:
		return "85-89"
	case score >= 90 && score <= 100:
		return "90-100"
	default:
		return "OUT_OF_RANGE"
	}
}

func Reliability(n int) string {
	switch {
	case n < 30:
		return "INSUFFICIENT"
	case n < 100:
		return "PRELIMINARY"
	case n < 500:
		return "MODERATE"
	default:
		return "STRONGER_EVIDENCE"
	}
}

func Build(samples []Sample) Report {
	report := Report{StatusCounts: map[Status]int{}, CalculationTimestamp: time.Now().UTC(), ReliabilityDefinition: "<30 insufficient; 30-99 preliminary; 100-499 moderate; >=500 stronger evidence", CumulativeGross: []float64{}, CumulativeNet: []float64{}, Warnings: []string{}}
	var evaluated []Sample
	for _, sample := range samples {
		report.StatusCounts[sample.SimulationStatus]++
		if sample.GrossReturn != nil && sample.NetReturn != nil && sample.SimulationStatus != IncompleteSimulation && sample.SimulationStatus != PendingSimulation {
			evaluated = append(evaluated, sample)
		}
	}
	report.Reliability = Reliability(len(evaluated))
	report.Metrics = metrics(evaluated, report.StatusCounts)
	report.EdgeScore = edge(evaluated, report.Reliability)
	report.Breakdowns = breakdowns(evaluated)
	report.Horizons = horizons(evaluated)
	for _, sample := range evaluated {
		report.CumulativeGross = append(report.CumulativeGross, valueAt(report.CumulativeGross, *sample.GrossReturn))
		report.CumulativeNet = append(report.CumulativeNet, valueAt(report.CumulativeNet, *sample.NetReturn))
	}
	report.Drawdown = drawdown(report.CumulativeNet)
	if report.Reliability == "INSUFFICIENT" {
		report.Warnings = append(report.Warnings, "INSUFFICIENT_SAMPLE")
	}
	if report.StatusCounts[IncompleteSimulation] > 0 {
		report.Warnings = append(report.Warnings, "INCOMPLETE_SIMULATION", "INCOMPLETE_OUTCOME")
	}
	for _, s := range evaluated {
		if s.EntryFee == 0 || s.ExitFee == 0 {
			report.Warnings = append(report.Warnings, "MISSING_FEE")
			break
		}
	}
	for _, s := range evaluated {
		if s.EntrySlippage == 0 || s.ExitSlippage == 0 {
			report.Warnings = append(report.Warnings, "MISSING_SLIPPAGE")
			break
		}
	}
	if !hasHorizonSamples(report.Horizons) {
		report.Warnings = append(report.Warnings, "RETURN_HORIZON_NET_SIMULATION_UNAVAILABLE")
	}
	return report
}

func valueAt(series []float64, value float64) float64 {
	if len(series) == 0 {
		return value
	}
	return series[len(series)-1] + value
}
func ptrMetric(name, definition, unit string, value float64, count int) Metric {
	return Metric{Name: name, Definition: definition, Unit: unit, Value: value, SampleCount: count}
}
func metrics(samples []Sample, statuses map[Status]int) []Metric {
	n := len(samples)
	gross, net, mfe, mae, fees, exitFees, entrySlip, exitSlip, duration := []float64{}, []float64{}, []float64{}, []float64{}, 0.0, 0.0, 0.0, 0.0, 0.0
	wins, targets, invalidations := 0, 0, 0
	for _, s := range samples {
		gross = append(gross, *s.GrossReturn)
		net = append(net, *s.NetReturn)
		mfe = append(mfe, s.MFE)
		mae = append(mae, s.MAE)
		fees += s.EntryFee
		exitFees += s.ExitFee
		entrySlip += s.EntrySlippage
		exitSlip += s.ExitSlippage
		duration += s.Duration.Seconds()
		if *s.NetReturn > 0 {
			wins++
		}
		if s.TargetHit {
			targets++
		}
		if s.InvalidationHit {
			invalidations++
		}
	}
	winRate, precision, targetRate, invalidationRate := ratio(float64(wins), n), ratio(float64(wins), n), ratio(float64(targets), n), ratio(float64(invalidations), n)
	profit, loss := 0.0, 0.0
	for _, v := range net {
		if v > 0 {
			profit += v
		} else {
			loss -= v
		}
	}
	profitFactor := 0.0
	if loss > 0 {
		profitFactor = profit / loss
	}
	maxDD := 0.0
	for _, v := range drawdown(cumulative(net)) {
		if v < maxDD {
			maxDD = v
		}
	}
	return []Metric{
		ptrMetric("evaluated_signals", "Simulations with complete gross and net execution return", "count", float64(n), n), ptrMetric("pending_signals", "Signals without an execution simulation", "count", float64(statuses[PendingSimulation]), 0), ptrMetric("incomplete_simulations", "Simulations without a valid execution outcome; excluded from returns", "count", float64(statuses[IncompleteSimulation]), 0), ptrMetric("partial_fill_simulations", "Simulations with partial execution", "count", float64(statuses[PartialFill]), 0),
		ptrMetric("win_rate", "Share of evaluated simulations with positive net return", "decimal", winRate, n), ptrMetric("precision", "Positive net-return precision", "decimal", precision, n), ptrMetric("average_gross_return", "Mean simulated gross return before costs", "decimal", mean(gross), n), ptrMetric("average_net_return", "Mean simulated net return after fees and slippage", "decimal", mean(net), n), ptrMetric("median_gross_return", "Median simulated gross return", "decimal", median(gross), n), ptrMetric("median_net_return", "Median simulated net return", "decimal", median(net), n),
		ptrMetric("gross_expectancy", "Mean gross return per evaluated simulation", "decimal", mean(gross), n), ptrMetric("net_expectancy", "Mean net return per evaluated simulation", "decimal", mean(net), n), ptrMetric("profit_factor", "Gross positive net returns divided by absolute gross negative net returns", "ratio", profitFactor, n), ptrMetric("maximum_drawdown", "Largest cumulative simulated net-return decline", "decimal", maxDD, n), ptrMetric("mfe", "Mean maximum favorable excursion", "decimal", mean(mfe), n), ptrMetric("mae", "Mean maximum adverse excursion", "decimal", mean(mae), n),
		ptrMetric("target_hit_rate", "Target-hit share of evaluated simulations", "decimal", targetRate, n), ptrMetric("invalidation_rate", "Invalidation-hit share of evaluated simulations", "decimal", invalidationRate, n), ptrMetric("average_signal_duration", "Mean duration from creation to simulation outcome", "seconds", ratio(duration, n), n), ptrMetric("entry_fee_impact", "Total entry fee paid", "USDT", fees, n), ptrMetric("exit_fee_impact", "Total exit fee paid", "USDT", exitFees, n), ptrMetric("entry_slippage_impact", "Total entry slippage cost", "USDT", entrySlip, n), ptrMetric("exit_slippage_impact", "Total exit slippage cost", "USDT", exitSlip, n), ptrMetric("total_transaction_cost", "Fees plus slippage", "USDT", fees+exitFees+entrySlip+exitSlip, n),
	}
}
func ratio(n float64, d int) float64 {
	if d == 0 {
		return 0
	}
	return n / float64(d)
}
func mean(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return ratio(sum, len(values))
}
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	c := append([]float64(nil), values...)
	sort.Float64s(c)
	mid := len(c) / 2
	if len(c)%2 == 0 {
		return (c[mid-1] + c[mid]) / 2
	}
	return c[mid]
}
func cumulative(values []float64) []float64 {
	r := make([]float64, 0, len(values))
	for _, v := range values {
		r = append(r, valueAt(r, v))
	}
	return r
}
func drawdown(values []float64) []float64 {
	r := make([]float64, len(values))
	peak := 0.0
	for i, v := range values {
		if v > peak {
			peak = v
		}
		r[i] = v - peak
	}
	return r
}
func breakdowns(samples []Sample) []Breakdown {
	dimensions := map[string]func(Sample) string{"notional": func(s Sample) string { return formatNotional(s.Notional) }, "pair": func(s Sample) string { return s.Pair }, "tier": func(s Sample) string { return s.Tier }, "timeframe": func(s Sample) string { return s.Timeframe }, "score_bucket": func(s Sample) string { return ScoreBucket(s.Score) }, "market_regime": func(s Sample) string { return s.MarketRegime }, "rule_version": func(s Sample) string { return s.RuleVersion }, "model_version": func(s Sample) string { return s.ModelVersion }, "ai_decision": func(s Sample) string { return s.AIDecision }, "ai_provider": func(s Sample) string { return s.AIProvider }, "data_quality_bucket": func(s Sample) string { return s.DataQuality }, "spoof_risk_bucket": func(s Sample) string { return s.SpoofRisk }, "signal_status": func(s Sample) string { return s.SignalStatus }}
	out := []Breakdown{}
	for dimension, get := range dimensions {
		groups := map[string][]Sample{}
		for _, s := range samples {
			groups[get(s)] = append(groups[get(s)], s)
		}
		for value, items := range groups {
			g, n, w := 0.0, 0.0, 0
			for _, s := range items {
				g += *s.GrossReturn
				n += *s.NetReturn
				if *s.NetReturn > 0 {
					w++
				}
			}
			out = append(out, Breakdown{dimension, value, len(items), ratio(g, len(items)), ratio(n, len(items)), ratio(float64(w), len(items))})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dimension == out[j].Dimension {
			return out[i].Value < out[j].Value
		}
		return out[i].Dimension < out[j].Dimension
	})
	return out
}

func horizons(samples []Sample) []Horizon {
	ordered := []string{"1m", "5m", "15m", "30m", "1h", "4h", "8h", "24h"}
	out := make([]Horizon, 0, len(ordered))
	for _, name := range ordered {
		gross, net := []float64{}, []float64{}
		positive := 0
		for _, sample := range samples {
			value, ok := sample.HorizonReturns[name]
			if !ok {
				continue
			}
			gross = append(gross, value.GrossReturn)
			net = append(net, value.NetReturn)
			if value.NetReturn > 0 {
				positive++
			}
		}
		out = append(out, Horizon{Horizon: name, MeanGrossReturn: mean(gross), MedianGrossReturn: median(gross), MeanNetReturn: mean(net), MedianNetReturn: median(net), PositiveRate: ratio(float64(positive), len(net)), SampleCount: len(net), ConfidenceInterval: confidenceInterval(net)})
	}
	return out
}

func confidenceInterval(values []float64) *[2]float64 {
	if len(values) < 2 {
		return nil
	}
	avg := mean(values)
	variance := 0.0
	for _, value := range values {
		variance += (value - avg) * (value - avg)
	}
	stdErr := math.Sqrt(variance/float64(len(values)-1)) / math.Sqrt(float64(len(values)))
	ci := [2]float64{avg - 1.96*stdErr, avg + 1.96*stdErr}
	return &ci
}

func hasHorizonSamples(horizons []Horizon) bool {
	for _, horizon := range horizons {
		if horizon.SampleCount > 0 {
			return true
		}
	}
	return false
}
func formatNotional(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
func edge(samples []Sample, reliability string) EdgeScore {
	n := len(samples)
	netExp, pf, precision, dd, calibrationError := 0.0, 0.0, 0.0, 0.0, 0.0
	if n > 0 {
		net := make([]float64, 0, n)
		for _, s := range samples {
			net = append(net, *s.NetReturn)
			outcome := 0.0
			if *s.NetReturn > 0 {
				outcome = 1
			}
			calibrationError += math.Abs(clamp(s.Score/100, 0, 1) - outcome)
		}
		netExp = clamp(mean(net)*10000, 0, 100)
		profit, loss := 0.0, 0.0
		for _, v := range net {
			if v > 0 {
				profit += v
			} else {
				loss -= v
			}
		}
		if loss > 0 {
			pf = clamp(profit/loss*25, 0, 100)
		}
		for _, v := range net {
			if v > 0 {
				precision++
			}
		}
		precision = precision / float64(n) * 100
		for _, v := range drawdown(cumulative(net)) {
			if v < dd {
				dd = v
			}
		}
		dd = clamp(100+dd*10000, 0, 100)
	}
	calibration := 100 - (calibrationError / float64(maxInt(n, 1)) * 100)
	reliabilityScore := map[string]float64{"INSUFFICIENT": 0, "PRELIMINARY": 35, "MODERATE": 70, "STRONGER_EVIDENCE": 100}[reliability]
	parts := []EdgeComponent{{"net_expectancy", .30, netExp, 0}, {"profit_factor", .20, pf, 0}, {"precision", .15, precision, 0}, {"score_calibration", .15, calibration, 0}, {"drawdown_control", .10, dd, 0}, {"sample_reliability", .10, reliabilityScore, 0}}
	score := 0.0
	for i := range parts {
		parts[i].Contribution = parts[i].Weight * parts[i].Score
		score += parts[i].Contribution
	}
	return EdgeScore{score, parts}
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

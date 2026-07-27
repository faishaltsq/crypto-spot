package threshold

import "testing"

func TestCalculate(t *testing.T) {
	cfg := DefaultConfig(80)
	cases := []struct {
		name  string
		input Input
		want  float64
		block bool
	}{
		{"tier A", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid, ActualScore: 80}, 80, false},
		{"tier B", Input{Tier: TierB, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid, ActualScore: 83}, 83, false},
		{"tier C", Input{Tier: TierC, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid, ActualScore: 87}, 87, false},
		{"strong uptrend", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 80, false},
		{"weak uptrend", Input{Tier: TierA, Regime: WeakUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 82, false},
		{"ranging", Input{Tier: TierA, Regime: Ranging, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 84, false},
		{"high volatility regime", Input{Tier: TierA, Regime: HighVolatility, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 85, false},
		{"market sell-off", Input{Tier: TierA, Regime: MarketSellOff, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 90, false},
		{"pump", Input{Tier: TierA, Regime: PumpCondition, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 88, false},
		{"moderate spoof", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofModerate, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 84, false},
		{"high spoof blocks", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofHigh, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 80, true},
		{"moderate liquidity", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityModerate, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 83, false},
		{"low liquidity blocks", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityLow, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 80, true},
		{"degraded data blocks", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityDegraded}, 80, true},
		{"stale data blocks", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityStale}, 80, true},
		{"correlation", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationHighCluster, DataQuality: DataQualityValid}, 85, false},
		{"combined", Input{Tier: TierB, Regime: HighVolatility, VolatilityPercentile: 90, SpoofRisk: SpoofModerate, Liquidity: LiquidityModerate, Correlation: CorrelationModerateBurst, DataQuality: DataQualityValid, ActualScore: 91}, 99, false},
		{"missing regime", Input{Tier: TierA, VolatilityPercentile: 10, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 85, false},
		{"missing volatility", Input{Tier: TierA, Regime: StrongUptrend, VolatilityPercentile: -1, SpoofRisk: SpoofLow, Liquidity: LiquidityHealthy, Correlation: CorrelationIndependent, DataQuality: DataQualityValid}, 80, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Calculate(cfg, tc.input)
			if got.FinalThreshold != tc.want || got.Blocked != tc.block {
				t.Fatalf("threshold=%v blocked=%v, want threshold=%v blocked=%v", got.FinalThreshold, got.Blocked, tc.want, tc.block)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig(80)
	cfg.Volatility = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected config validation error")
	}
}

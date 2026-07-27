package threshold

import "fmt"

func (c Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("threshold version is required")
	}
	if c.BaseThreshold < 0 || c.BaseThreshold > 100 {
		return fmt.Errorf("base threshold must be between 0 and 100")
	}
	if c.Tier == nil || c.Regime == nil || c.Spoof == nil || c.Liquidity == nil || c.Correlation == nil || len(c.Volatility) == 0 {
		return fmt.Errorf("all adjustment groups are required")
	}
	last := -1.0
	for _, v := range c.Volatility {
		if v.MinPercentile < 0 || v.MinPercentile > 100 || v.MinPercentile <= last {
			return fmt.Errorf("volatility bands must be strictly ascending percentiles between 0 and 100")
		}
		last = v.MinPercentile
	}
	return nil
}

package execution_simulation

func fee(amount, bps float64, enabled bool) *float64 {
	if !enabled || amount <= 0 {
		return nil
	}
	value := amount * bps / 10000
	return &value
}

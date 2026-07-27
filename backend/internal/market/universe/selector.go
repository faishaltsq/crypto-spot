package universe

type TierLimits struct {
	LimitA int
	LimitB int
	LimitC int
}

func AssignTiers(ranked []RankedPair, limits TierLimits) []RankedPair {
	limitA := limits.LimitA
	limitB := limitA + limits.LimitB
	limitC := limitB + limits.LimitC

	var active []RankedPair
	for i, p := range ranked {
		if i < limitA {
			p.Tier = 1
			p.Qualified = true
		} else if i < limitB {
			p.Tier = 2
			p.Qualified = true
		} else if i < limitC {
			p.Tier = 3
			p.Qualified = true
		} else {
			p.Tier = 0
			p.Qualified = false
			p.RejectionReason = "Out of limits"
		}
		
		if p.Qualified {
			active = append(active, p)
		} else {
			// We can break early if we just want to return active pairs, 
			// but keeping it simple and just returning active.
		}
	}
	return active
}

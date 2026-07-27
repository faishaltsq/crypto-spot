package quality

// ScoreToStatus maps a numeric quality score to its status category.
//
//	90-100: VALID
//	75-89:  DEGRADED
//	50-74:  STALE (data may be partially usable but signal should be cautious)
//	0-49:   BLOCKED (no signals allowed)
func ScoreToStatus(score float64) QualityStatus {
	switch {
	case score >= 90:
		return StatusValid
	case score >= 75:
		return StatusDegraded
	case score >= 50:
		return StatusStale
	default:
		return StatusBlocked
	}
}

// ComputeScore runs all quality rules and returns the final score plus
// the list of failed reason codes and individual rule results.
func ComputeScore(input PairHealthInput, cfg QualityConfig) (float64, []ReasonCode, []RuleResult) {
	rules := AllRules()
	score := 100.0
	var failedReasons []ReasonCode
	results := make([]RuleResult, 0, len(rules))

	for _, rule := range rules {
		passed, detail := rule.Check(input, cfg)
		result := RuleResult{
			Rule:    string(rule.Code),
			Code:    rule.Code,
			Passed:  passed,
			Score:   100.0 - rule.Penalty, // score this rule contributes if passed
			Penalty: 0,
			Reason:  detail,
		}
		if !passed {
			result.Penalty = rule.Penalty
			result.Score = 0
			score -= rule.Penalty
			failedReasons = append(failedReasons, rule.Code)
		}
		results = append(results, result)
	}

	// Clamp score to [0, 100]
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, failedReasons, results
}

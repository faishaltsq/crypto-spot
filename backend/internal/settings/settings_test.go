package settings

import "testing"

func TestValidateRejectsInvalidCrossFieldSettings(t *testing.T) {
	tests := []map[string]interface{}{
		{"signal_setup_score": float64(80), "signal_confirm_score": float64(70)},
		{"minimum_model_probability": float64(1.1)},
		{"market_pair_limit": float64(10), "tier_a_limit": float64(6), "tier_b_limit": float64(5), "tier_c_limit": float64(0)},
		{"paper_simulation_notionals": []interface{}{float64(0)}},
	}
	for _, input := range tests {
		if err := Validate(input); err == nil {
			t.Fatalf("expected validation failure for %#v", input)
		}
	}
}

func TestMetadataClassifiesReloadBehavior(t *testing.T) {
	if mode, _ := Metadata("notification_rate_limit"); mode != Restart {
		t.Fatalf("got %s", mode)
	}
	if mode, _ := Metadata("market_pair_limit"); mode != Restart {
		t.Fatalf("got %s", mode)
	}
	if mode, _ := Metadata("ai_provider"); mode != Restart {
		t.Fatalf("got %s", mode)
	}
}

func TestValidateRejectsSecretLikeUnknownKeys(t *testing.T) {
	for _, key := range []string{"database_password", "redis_password", "jwt_secret", "api_key", "internal_token"} {
		if err := Validate(map[string]interface{}{key: "leak"}); err == nil {
			t.Fatalf("expected %s to be rejected", key)
		}
	}
}

func TestValidateRejectsNonNumericThreshold(t *testing.T) {
	if err := Validate(map[string]interface{}{"signal_setup_score": "70"}); err == nil {
		t.Fatal("expected string score rejection")
	}
}

func TestValidatePreferencesRejectsSecretKey(t *testing.T) {
	if err := ValidatePreferences(map[string]interface{}{"api_key": "leak"}); err == nil {
		t.Fatal("expected secret-like preference rejection")
	}
}

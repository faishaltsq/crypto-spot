package settings

import (
	"fmt"
	"math"
	"sort"
)

type ReloadMode string

const (
	Immediate      ReloadMode = "immediate"
	NewSignal      ReloadMode = "new_signal"
	Resubscription ReloadMode = "resubscription"
	Restart        ReloadMode = "restart"
)

type Value struct {
	Key             string      `json:"key"`
	Value           interface{} `json:"value"`
	ReloadMode      ReloadMode  `json:"reloadMode"`
	RestartRequired bool        `json:"restartRequired"`
}

type AuditLog struct {
	SettingKey      string      `json:"settingKey"`
	PreviousValue   interface{} `json:"previousValue"`
	NewValue        interface{} `json:"newValue"`
	ChangedBy       string      `json:"changedBy"`
	ChangedAt       string      `json:"changedAt"`
	Reason          string      `json:"reason"`
	Result          string      `json:"result"`
	EffectiveAt     *string     `json:"effectiveAt"`
	RestartRequired bool        `json:"restartRequired"`
}

var metadata = map[string]ReloadMode{
	// Services currently bind configuration during startup. Keep overrides
	// restart-gated until an owning service supports live reload.
	"market_pair_limit":           Restart,
	"tier_a_limit":                Restart,
	"tier_b_limit":                Restart,
	"tier_c_limit":                Restart,
	"universe_refresh_interval":   Restart,
	"signal_setup_score":          Restart,
	"signal_confirm_score":        Restart,
	"minimum_rule_score":          Restart,
	"minimum_model_probability":   Restart,
	"minimum_data_quality":        Restart,
	"maximum_spoof_score":         Restart,
	"global_active_signal_limit":  Restart,
	"pair_active_signal_limit":    Restart,
	"cluster_active_signal_limit": Restart,
	"pair_cooldown_minutes":       Restart,
	"notification_rate_limit":     Restart,
	"recorder_retention_days":     Restart,
	"active_rule_version":         Restart,
	"active_model_version":        Restart,
	"ai_provider":                 Restart,
	"paper_simulation_notionals":  Restart,
}

var preferenceKeys = map[string]struct{}{
	"theme": {}, "density": {}, "fontSize": {}, "reduceMotion": {}, "numberFormat": {}, "timezone": {}, "currencyDisplay": {}, "positiveColor": {}, "negativeColor": {}, "chartWatermark": {}, "decimalPrecision": {},
	"showLeftPairPanel": {}, "showRightSignalPanel": {}, "showDiagnostic": {}, "showOrderBook": {}, "showRecentTrades": {}, "defaultPanelSizes": {}, "defaultPair": {}, "defaultTimeframe": {}, "defaultTerminalRoute": {},
	"chartProvider": {}, "chartType": {}, "defaultIndicators": {}, "signalOverlay": {}, "drawingPersistence": {}, "autoScale": {}, "logarithmicScale": {},
	"browserNotifications": {}, "sound": {}, "minimumSignalScore": {}, "allowedSignalStates": {}, "allowedRiskLevels": {}, "allowedTimeframes": {}, "maximumNotificationsPerHour": {}, "digestMode": {}, "quietHours": {}, "systemAlerts": {}, "aiAlerts": {}, "staleDataAlerts": {}, "orderBookResyncAlerts": {},
	"defaultWatchlist": {}, "watchlistTimeframe": {}, "defaultAlertScore": {}, "defaultRiskFilter": {},
}

func Metadata(key string) (ReloadMode, bool) {
	mode, ok := metadata[key]
	return mode, ok
}

func Keys() []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ValidatePreferences(values map[string]interface{}) error {
	for key := range values {
		if _, ok := preferenceKeys[key]; !ok {
			return fmt.Errorf("unsupported preference key: %s", key)
		}
	}
	return nil
}

func Validate(values map[string]interface{}) error {
	for key := range values {
		if _, ok := Metadata(key); !ok {
			return fmt.Errorf("unsupported setting_key: %s", key)
		}
	}
	setup, hasSetup := number(values, "signal_setup_score")
	confirm, hasConfirm := number(values, "signal_confirm_score")
	if _, exists := values["signal_setup_score"]; exists && !hasSetup {
		return fmt.Errorf("signal_setup_score must be a number")
	}
	if _, exists := values["signal_confirm_score"]; exists && !hasConfirm {
		return fmt.Errorf("signal_confirm_score must be a number")
	}
	if hasSetup && (setup < 0 || setup > 100) {
		return fmt.Errorf("signal_setup_score must be between 0 and 100")
	}
	if hasConfirm && (confirm < 0 || confirm > 100) {
		return fmt.Errorf("signal_confirm_score must be between 0 and 100")
	}
	if hasSetup && hasConfirm && confirm < setup {
		return fmt.Errorf("signal_confirm_score must be greater than or equal to signal_setup_score")
	}
	if _, exists := values["minimum_model_probability"]; exists {
		value, ok := number(values, "minimum_model_probability")
		if !ok || math.IsNaN(value) || value < 0 || value > 1 {
			return fmt.Errorf("minimum_model_probability must be between 0 and 1")
		}
	}
	for _, key := range []string{"minimum_data_quality", "maximum_spoof_score"} {
		if _, exists := values[key]; exists {
			value, ok := number(values, key)
			if !ok || math.IsNaN(value) || value < 0 || value > 100 {
				return fmt.Errorf("%s must be between 0 and 100", key)
			}
		}
	}
	for _, key := range []string{"market_pair_limit", "tier_a_limit", "tier_b_limit", "tier_c_limit", "global_active_signal_limit", "pair_active_signal_limit", "cluster_active_signal_limit", "pair_cooldown_minutes", "notification_rate_limit", "recorder_retention_days", "universe_refresh_interval"} {
		if _, exists := values[key]; exists {
			value, ok := number(values, key)
			if !ok || value < 0 || math.Trunc(value) != value {
				return fmt.Errorf("%s must be a non-negative integer", key)
			}
		}
	}
	if limit, ok := number(values, "market_pair_limit"); ok {
		tierTotal := valueOrZero(values, "tier_a_limit") + valueOrZero(values, "tier_b_limit") + valueOrZero(values, "tier_c_limit")
		if tierTotal > limit {
			return fmt.Errorf("tier limits must not exceed market_pair_limit")
		}
	}
	if notionals, exists := values["paper_simulation_notionals"]; exists {
		items, ok := notionals.([]interface{})
		if !ok || len(items) == 0 {
			return fmt.Errorf("paper_simulation_notionals must be a non-empty array")
		}
		for _, item := range items {
			number, ok := item.(float64)
			if !ok || number <= 0 {
				return fmt.Errorf("paper_simulation_notionals must contain positive numbers")
			}
		}
	}
	return nil
}

func number(values map[string]interface{}, key string) (float64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	number, ok := value.(float64)
	return number, ok
}
func valueOrZero(values map[string]interface{}, key string) float64 {
	value, _ := number(values, key)
	return value
}

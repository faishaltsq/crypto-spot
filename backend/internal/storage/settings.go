package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/crypto-spot-signal/internal/settings"
)

type SettingVersion struct {
	Version   int                    `json:"version"`
	Settings  map[string]interface{} `json:"settings"`
	ChangedBy string                 `json:"changedBy"`
	Reason    string                 `json:"reason"`
	CreatedAt time.Time              `json:"createdAt"`
}

func (r *Repository) ListSystemSettings(ctx context.Context) ([]settings.Value, error) {
	rows, err := r.pool.Query(ctx, `SELECT setting_key, setting_value, reload_mode, restart_required FROM system_settings ORDER BY setting_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []settings.Value{}
	for rows.Next() {
		var item settings.Value
		var raw []byte
		if err := rows.Scan(&item.Key, &raw, &item.ReloadMode, &item.RestartRequired); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Value); err != nil {
			return nil, fmt.Errorf("decode setting %s: %w", item.Key, err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListSystemSettingValues(ctx context.Context) (map[string]interface{}, error) {
	items, err := r.ListSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	values := make(map[string]interface{}, len(items))
	for _, item := range items {
		values[item.Key] = item.Value
	}
	return values, nil
}

func (r *Repository) SaveSystemSettings(ctx context.Context, changes map[string]interface{}, changedBy, reason string) ([]settings.Value, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current := map[string]interface{}{}
	rows, err := tx.Query(ctx, `SELECT setting_key, setting_value FROM system_settings FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		var raw []byte
		var value interface{}
		if err := rows.Scan(&key, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			rows.Close()
			return nil, err
		}
		current[key] = value
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	previousValues := make(map[string]interface{}, len(current))
	for key, value := range current {
		previousValues[key] = value
	}
	for key, value := range changes {
		current[key] = value
	}
	if err := settings.Validate(current); err != nil {
		return nil, err
	}
	var version int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM system_setting_versions`).Scan(&version); err != nil {
		return nil, err
	}
	snapshot, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO system_setting_versions (version, settings, changed_by, reason) VALUES ($1, $2, $3, $4)`, version, snapshot, changedBy, reason); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	result := make([]settings.Value, 0, len(changes))
	for key, value := range changes {
		mode, _ := settings.Metadata(key)
		restart := mode == settings.Restart
		previous, previousExists := previousValues[key]
		payload, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO system_settings (setting_key, setting_value, reload_mode, restart_required, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (setting_key) DO UPDATE SET setting_value = EXCLUDED.setting_value, reload_mode = EXCLUDED.reload_mode, restart_required = EXCLUDED.restart_required, updated_at = NOW()`, key, payload, mode, restart); err != nil {
			return nil, err
		}
		var previousJSON []byte
		if previousExists {
			previousJSON, _ = json.Marshal(previous)
		}
		effective := now
		if restart {
			effective = time.Time{}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO system_setting_audit_logs (setting_key, previous_value, new_value, changed_by, changed_at, reason, result, effective_at, restart_required) VALUES ($1, $2, $3, $4, $5, $6, 'APPLIED', $7, $8)`, key, previousJSON, payload, changedBy, now, reason, nullableTime(effective), restart); err != nil {
			return nil, err
		}
		result = append(result, settings.Value{Key: key, Value: value, ReloadMode: mode, RestartRequired: restart})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) ResetSystemSettings(ctx context.Context, changedBy, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT setting_key, setting_value, restart_required FROM system_settings FOR UPDATE`)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	type resetItem struct {
		key     string
		value   []byte
		restart bool
	}
	items := []resetItem{}
	for rows.Next() {
		var key string
		var value []byte
		var restart bool
		if err := rows.Scan(&key, &value, &restart); err != nil {
			rows.Close()
			return err
		}
		items = append(items, resetItem{key, value, restart})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, item := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO system_setting_audit_logs (setting_key, previous_value, new_value, changed_by, changed_at, reason, result, effective_at, restart_required) VALUES ($1, $2, 'null'::jsonb, $3, $4, $5, 'RESET', NULL, $6)`, item.key, item.value, changedBy, now, reason, item.restart); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM system_settings`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}

func (r *Repository) ListSettingVersions(ctx context.Context, limit int) ([]SettingVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `SELECT version, settings, changed_by, reason, created_at FROM system_setting_versions ORDER BY version DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SettingVersion{}
	for rows.Next() {
		var item SettingVersion
		var raw []byte
		if err := rows.Scan(&item.Version, &raw, &item.ChangedBy, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Settings); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListSettingAuditLogs(ctx context.Context, limit int) ([]settings.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `SELECT setting_key, previous_value, new_value, changed_by, changed_at, reason, result, effective_at, restart_required FROM system_setting_audit_logs ORDER BY changed_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []settings.AuditLog{}
	for rows.Next() {
		var item settings.AuditLog
		var previous, next []byte
		var effective *time.Time
		var changed time.Time
		if err := rows.Scan(&item.SettingKey, &previous, &next, &item.ChangedBy, &changed, &item.Reason, &item.Result, &effective, &item.RestartRequired); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(previous, &item.PreviousValue)
		if err := json.Unmarshal(next, &item.NewValue); err != nil {
			return nil, err
		}
		item.ChangedAt = changed.UTC().Format(time.RFC3339)
		if effective != nil {
			text := effective.UTC().Format(time.RFC3339)
			item.EffectiveAt = &text
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveUserPreferences(ctx context.Context, userID string, preferences map[string]interface{}) error {
	payload, err := json.Marshal(preferences)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO user_preferences (user_id, preferences, updated_at) VALUES ($1::uuid, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET preferences = EXCLUDED.preferences, updated_at = NOW()`, userID, payload)
	return err
}

func (r *Repository) UserPreferences(ctx context.Context, userID string) (map[string]interface{}, bool, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT preferences FROM user_preferences WHERE user_id = $1::uuid`, userID).Scan(&raw)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return map[string]interface{}{}, false, nil
		}
		return nil, false, err
	}
	result := map[string]interface{}{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false, err
	}
	return result, true, nil
}

package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const legacyVersion = 5

type Config struct {
	DatabaseURL string
	SourceDir   string
	Timeout     time.Duration
	Retries     int
	RetryDelay  time.Duration
}

type Status struct {
	Version uint
	Dirty   bool
	Latest  uint
}

func DefaultConfig(databaseURL string) Config {
	return Config{
		DatabaseURL: databaseURL,
		SourceDir:   envOr("MIGRATIONS_DIR", "migrations/versioned"),
		Timeout:     envDurationSeconds("MIGRATION_TIMEOUT_SECONDS", 60),
		Retries:     envInt("MIGRATION_RETRIES", 30),
		RetryDelay:  envDurationSeconds("MIGRATION_RETRY_DELAY_SECONDS", 2),
	}
}

func Up(ctx context.Context, cfg Config) (Status, error) {
	m, err := open(ctx, cfg, true)
	if err != nil {
		return Status{}, err
	}
	defer m.Close()
	if err := runWithTimeout(ctx, cfg.Timeout, m.Up); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return Status{}, fmt.Errorf("apply migrations: %w", err)
	}
	return status(m, cfg.SourceDir)
}

func Down(ctx context.Context, cfg Config) (Status, error) {
	legacy, err := isLegacySchema(ctx, cfg.DatabaseURL)
	if err != nil {
		return Status{}, err
	}
	if legacy {
		return Status{}, errors.New("legacy schema has no migration history; run up before rollback")
	}
	m, err := open(ctx, cfg, false)
	if err != nil {
		return Status{}, err
	}
	defer m.Close()
	current, dirty, err := m.Version()
	if err != nil {
		return Status{}, fmt.Errorf("read migration version: %w", err)
	}
	if dirty {
		return Status{}, fmt.Errorf("migration database is dirty at version %d; repair before rollback", current)
	}
	if current != latestVersion(cfg.SourceDir) {
		return Status{}, fmt.Errorf("rollback only supports latest migration %d; current version is %d", latestVersion(cfg.SourceDir), current)
	}
	if err := runWithTimeout(ctx, cfg.Timeout, func() error { return m.Steps(-1) }); err != nil {
		return Status{}, fmt.Errorf("rollback migration %d: %w", current, err)
	}
	return status(m, cfg.SourceDir)
}

func StatusOf(ctx context.Context, cfg Config) (Status, error) {
	legacy, err := isLegacySchema(ctx, cfg.DatabaseURL)
	if err != nil {
		return Status{}, err
	}
	if legacy {
		return Status{Version: legacyVersion, Latest: latestVersion(cfg.SourceDir)}, nil
	}
	m, err := open(ctx, cfg, false)
	if err != nil {
		return Status{}, err
	}
	defer m.Close()
	return status(m, cfg.SourceDir)
}

// Repair clears golang-migrate's dirty marker at its recorded version after the failed migration is fixed.
func Repair(ctx context.Context, cfg Config) (Status, error) {
	m, err := open(ctx, cfg, false)
	if err != nil {
		return Status{}, err
	}
	defer m.Close()
	version, dirty, err := m.Version()
	if err != nil {
		return Status{}, fmt.Errorf("read migration version: %w", err)
	}
	if !dirty {
		return Status{}, errors.New("migration database is not dirty")
	}
	if version == 0 || version > latestVersion(cfg.SourceDir) {
		return Status{}, fmt.Errorf("cannot repair invalid migration version %d", version)
	}
	if err := m.Force(int(version)); err != nil {
		return Status{}, fmt.Errorf("clear dirty migration state: %w", err)
	}
	return status(m, cfg.SourceDir)
}

func open(ctx context.Context, cfg Config, baseline bool) (*migrate.Migrate, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.Timeout <= 0 || cfg.Retries < 1 || cfg.RetryDelay <= 0 {
		return nil, errors.New("invalid migration timeout or retry configuration")
	}
	if _, err := os.Stat(cfg.SourceDir); err != nil {
		return nil, fmt.Errorf("migration source unavailable: %w", err)
	}
	if _, err := migrationVersions(cfg.SourceDir); err != nil {
		return nil, err
	}
	if err := validateChecksums(cfg.SourceDir); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	if err := waitForDatabase(ctx, cfg); err != nil {
		return nil, err
	}
	if baseline {
		if err := baselineLegacy(ctx, cfg); err != nil {
			return nil, err
		}
	}
	m, err := migrate.New("file://"+filepath.ToSlash(cfg.SourceDir), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open migration runner: %w", err)
	}
	return m, nil
}

func waitForDatabase(ctx context.Context, cfg Config) error {
	for attempt := 1; attempt <= cfg.Retries; attempt++ {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err == nil {
			err = pool.Ping(ctx)
			pool.Close()
		}
		if err == nil {
			return nil
		}
		if ctx.Err() != nil || attempt == cfg.Retries {
			return fmt.Errorf("database unavailable after %d attempts: %w", attempt, err)
		}
		log.Printf("migration: database unavailable, retrying (%d/%d)", attempt, cfg.Retries)
		select {
		case <-ctx.Done():
			return fmt.Errorf("migration database wait timed out: %w", ctx.Err())
		case <-time.After(cfg.RetryDelay):
		}
	}
	return errors.New("unreachable database retry state")
}

func runWithTimeout(ctx context.Context, timeout time.Duration, operation func() error) error {
	result := make(chan error, 1)
	go func() { result <- operation() }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("migration timed out: %w", ctx.Err())
	case <-time.After(timeout):
		return fmt.Errorf("migration timed out after %s", timeout)
	}
}

func baselineLegacy(ctx context.Context, cfg Config) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect for legacy schema check: %w", err)
	}
	defer pool.Close()
	legacy, err := legacySchema(ctx, pool)
	if err != nil {
		return err
	}
	if !legacy {
		return nil
	}
	log.Printf("migration: legacy schema detected; baselining at version %d", legacyVersion)
	m, err := migrate.New("file://"+filepath.ToSlash(cfg.SourceDir), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open legacy baseline runner: %w", err)
	}
	defer m.Close()
	if err := m.Force(legacyVersion); err != nil {
		return fmt.Errorf("baseline legacy schema: %w", err)
	}
	return nil
}

func isLegacySchema(ctx context.Context, databaseURL string) (bool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return false, fmt.Errorf("connect for legacy schema check: %w", err)
	}
	defer pool.Close()
	return legacySchema(ctx, pool)
}

func legacySchema(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var historyExists, signalsExists bool
	var expectedTables, expectedColumns int
	if err := pool.QueryRow(ctx, `
		SELECT
			to_regclass('public.schema_migrations') IS NOT NULL,
			to_regclass('public.signals') IS NOT NULL,
			(SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND tablename = ANY(ARRAY[
				'candles', 'orderbook_metrics', 'market_features', 'signals', 'signal_outcomes', 'market_pairs', 'data_quality_history'
			])),
			(SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'signals' AND column_name = ANY(ARRAY[
				'signal_version', 'evidence', 'threshold_detail', 'data_quality_score', 'data_quality_status', 'data_source', 'missing_features', 'blocked_reasons'
			]))
	`).Scan(&historyExists, &signalsExists, &expectedTables, &expectedColumns); err != nil {
		return false, fmt.Errorf("check migration history: %w", err)
	}
	if historyExists || !signalsExists {
		return false, nil
	}
	if expectedTables != 7 || expectedColumns != 8 {
		return false, errors.New("legacy database has incomplete schema; refusing to baseline")
	}
	return true, nil
}

func validateChecksums(sourceDir string) error {
	manifest, err := os.ReadFile(filepath.Join(sourceDir, "checksums.sha256"))
	if err != nil {
		return fmt.Errorf("read migration checksum manifest: %w", err)
	}
	entries := strings.Fields(string(manifest))
	if len(entries) == 0 || len(entries)%2 != 0 {
		return errors.New("invalid migration checksum manifest")
	}
	listed := map[string]bool{}
	for i := 0; i < len(entries); i += 2 {
		expected, name := entries[i], entries[i+1]
		if strings.Contains(name, "/") || !strings.HasSuffix(name, ".sql") || listed[name] {
			return fmt.Errorf("invalid migration checksum entry: %s", name)
		}
		content, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			return fmt.Errorf("read migration for checksum: %w", err)
		}
		actual := sha256.Sum256(content)
		if expected != hex.EncodeToString(actual[:]) {
			return fmt.Errorf("migration checksum mismatch: %s", name)
		}
		listed[name] = true
	}
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("read migration source: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") && !listed[file.Name()] {
			return fmt.Errorf("migration missing from checksum manifest: %s", file.Name())
		}
	}
	return nil
}

func status(m *migrate.Migrate, sourceDir string) (Status, error) {
	latest := latestVersion(sourceDir)
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return Status{Latest: latest}, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("read migration version: %w", err)
	}
	return Status{Version: version, Dirty: dirty, Latest: latest}, nil
}

func migrationVersions(sourceDir string) ([]uint, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("read migration source: %w", err)
	}
	seen := map[uint]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		part := strings.SplitN(entry.Name(), "_", 2)[0]
		version64, err := strconv.ParseUint(part, 10, 32)
		if err != nil || version64 == 0 || seen[uint(version64)] {
			return nil, fmt.Errorf("invalid migration version sequence: %s", entry.Name())
		}
		version := uint(version64)
		if _, err := os.Stat(filepath.Join(sourceDir, strings.TrimSuffix(entry.Name(), ".up.sql")+".down.sql")); err != nil {
			return nil, fmt.Errorf("migration %d missing down file", version)
		}
		seen[version] = true
	}
	versions := make([]uint, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	for i, version := range versions {
		if version != uint(i+1) {
			return nil, fmt.Errorf("invalid migration version sequence: expected %d, found %d", i+1, version)
		}
	}
	return versions, nil
}

func latestVersion(sourceDir string) uint {
	versions, err := migrationVersions(sourceDir)
	if err != nil || len(versions) == 0 {
		return 0
	}
	return versions[len(versions)-1]
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(envOr(name, strconv.Itoa(fallback)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func envDurationSeconds(name string, fallback int) time.Duration {
	return time.Duration(envInt(name, fallback)) * time.Second
}

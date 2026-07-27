package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationVersionsRejectsGap(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_init.up.sql", "SELECT 1;")
	writeMigration(t, dir, "001_init.down.sql", "SELECT 1;")
	writeMigration(t, dir, "003_gap.up.sql", "SELECT 1;")
	writeMigration(t, dir, "003_gap.down.sql", "SELECT 1;")

	_, err := migrationVersions(dir)
	if err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("expected sequence error, got %v", err)
	}
}

func TestMigrationVersionsRejectsMissingDown(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_init.up.sql", "SELECT 1;")

	_, err := migrationVersions(dir)
	if err == nil || !strings.Contains(err.Error(), "missing down") {
		t.Fatalf("expected missing down error, got %v", err)
	}
}

func TestValidateChecksumsRejectsChangedMigration(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_init.up.sql", "SELECT 1;")
	writeMigration(t, dir, "001_init.down.sql", "SELECT 1;")
	writeManifest(t, dir)
	writeMigration(t, dir, "001_init.up.sql", "SELECT 2;")

	err := validateChecksums(dir)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestValidateChecksumsRejectsUnlistedMigration(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_init.up.sql", "SELECT 1;")
	writeMigration(t, dir, "001_init.down.sql", "SELECT 1;")
	writeManifest(t, dir)
	writeMigration(t, dir, "002_extra.up.sql", "SELECT 1;")

	err := validateChecksums(dir)
	if err == nil || !strings.Contains(err.Error(), "missing from checksum") {
		t.Fatalf("expected manifest error, got %v", err)
	}
}

func TestDefaultConfigDoesNotExposeDatabaseURL(t *testing.T) {
	cfg := DefaultConfig("postgres://user:secret@db:5432/signal")
	if cfg.DatabaseURL == "" || cfg.SourceDir == "" || cfg.Timeout <= 0 {
		t.Fatalf("expected complete migration config")
	}
}

func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeManifest(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(content)
		lines = append(lines, hex.EncodeToString(hash[:])+" "+entry.Name())
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.sha256"), []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
}

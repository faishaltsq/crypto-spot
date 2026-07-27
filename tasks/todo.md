# Migration Tasks

- [ ] Add versioned snapshots and runner validation.
  - Acceptance: New DB migrates to latest version; malformed sequence/checksum fails before execution.
  - Verify: Focused migration tests.
- [ ] Add command-line migration runner.
  - Acceptance: `up`, `down`, `status`, `version` return correct exit status and redact secrets.
  - Verify: `go test ./internal/migration ./cmd/migrate`.
- [ ] Wire Docker Compose lifecycle.
  - Acceptance: `migrate` waits for PostgreSQL health; backend waits for successful migration.
  - Verify: `docker compose config`.
- [ ] Add regression tests and documentation.
  - Acceptance: Legacy baseline, duplicate execution, dirty-state handling, Compose dependency checks covered.
  - Verify: Full Go test suite and isolated migration integration test.

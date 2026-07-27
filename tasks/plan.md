# Implementation Plan: Versioned Database Migrations

## Overview
Replace PostgreSQL init-script mounts with a one-shot `golang-migrate` service. The service records schema versions in `schema_migrations`, validates migration integrity before execution, and blocks backend startup until migrations complete.

## Architecture Decisions
- Use `golang-migrate` only. Its PostgreSQL driver provides version history and an advisory migration lock.
- Preserve `backend/migrations/*.sql` as legacy artifacts. Add immutable `.up.sql` and `.down.sql` snapshots in `backend/migrations/versioned/` for the runner.
- Detect legacy databases by the absence of `schema_migrations` plus existing application tables. Baseline those databases at version 5, then run repair migration 6.
- Permit `down` for only migration 6. Earlier versions are protected because they predate version tracking and contain foundational schema.
- Do not log database URLs. Errors identify operation and version only.

## Task List

### Phase 1: Runner Foundation
- [ ] Add immutable versioned migration snapshots and manifest validation.
- [ ] Add `cmd/migrate` commands: `up`, `down`, `status`, `version`.
- [ ] Verify checksum, sequence, retry, dirty-state, version, and exit behavior.

### Phase 2: Container Startup
- [ ] Build migration binary in backend image.
- [ ] Add one-shot `migrate` Compose service after PostgreSQL health.
- [ ] Make backend depend on migration completion.

### Phase 3: Verification
- [ ] Add migration unit/configuration tests.
- [ ] Run Go tests, migration binary build, Compose config validation, and isolated PostgreSQL integration tests when Docker is available.

## Risks and Mitigations
| Risk | Mitigation |
| --- | --- |
| Existing database has no schema history | Detect populated legacy schema, baseline at 5, run repair 6. |
| Previous init script dropped migration-004 tables | Migration 6 uses idempotent `CREATE TABLE IF NOT EXISTS` and indexes. |
| Concurrent migrators | PostgreSQL driver lock plus one Compose migration service. |
| Invalid or edited migration | SHA-256 manifest and ordered-version validation fail before DB writes. |

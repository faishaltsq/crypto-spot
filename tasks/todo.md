# Dynamic Signal Threshold Tasks

## Compare Evidence Key Fix 2026-07-27

- [x] Use unique keys for repeated evidence codes
  - Owner: `opencode-evidence-key-fix`
  - Started (UTC): `2026-07-27T13:50:00Z`
  - Files: `web/app/compare/page.tsx`, `web/tests/e2e.spec.ts`, `tasks/todo.md`
  - Verify: `cd web; npx playwright test tests/e2e.spec.ts; npm run build`
  - State: `done`
  - Completed (UTC): `2026-07-27T13:50:00Z`
  - Result: Evidence items now include occurrence index in React keys; repeated rule codes render without warnings.
  - Verify: focused desktop Playwright regression and `cd web; npm run build` passed.

## Repository Integration 2026-07-27

- [ ] Commit completed worktree changes
  - Owner: `opencode-integration`
  - Started (UTC): `2026-07-27T13:40:00Z`
  - Files: `.gitignore`, `tasks/todo.md`
  - Verify: `git diff --check; cd backend; go test ./...; cd web; npm run build`
  - State: `done`
  - Completed (UTC): `2026-07-27T13:40:00Z`
  - Result: staged completed Compare and Data Quality work, coordination docs, and binary ignores; excluded generated backend binaries.
  - Verify: `git diff --check`, `cd backend; go test ./...`, and `cd web; npm run build` passed.

## Compare Null Collection Fix 2026-07-27

- [x] Normalize nullable Compare API collections
  - Owner: `opencode-compare-fix`
  - Started (UTC): `2026-07-27T13:12:38Z`
  - Files: `web/app/compare/page.tsx`, `tasks/todo.md`
  - Verify: `cd web; npm run build`
  - State: `done`
  - Result: `null` API collection fields normalize to empty arrays before Compare rendering; scalar null still renders as `Partial`.
  - Verify: `cd web; npm run build` passed. Browser screenshot captured without the `.length` runtime exception.

## Data Quality Route 2026-07-27

- [x] Data Quality App Router page
  - Owner: `opencode-data-quality`
  - Started (UTC): `2026-07-27T12:38:35Z`
  - Files: `web/app/data-quality/page.tsx`, `web/lib/data-quality.ts`, `web/app/data-quality/loading.tsx`, `web/app/data-quality/error.tsx`, `web/components/terminal/TerminalHeader.tsx`, `web/components/GlobalRealtime.tsx`
  - Verify: `cd web; npm run build`
  - State: `done`
  - Result: `/data-quality` production route builds; quality API pagination/summary/history/reasons added; global WebSocket `quality.snapshot` refreshes page.
  - Verify: `cd backend; go test ./...` passed. `cd web; npm run build` passed.
  - Handoff: shared `server.go` has no pending Data Quality diff, but other active worktree changes remain. Do not stage unrelated files.

- [x] Threshold package and unit tests.
- [x] Threshold feature inputs.
- [x] Engine final-threshold decision integration.
- [x] Persist blocked audits and migration.
- [x] Signal Evidence breakdown and tests.
- [ ] Full verification and commit.

## Settings Integration 2026-07-27

- [ ] Settings endpoint, persistence, and UI integration
  - Owner: `opencode-settings`
  - Started (UTC): `2026-07-27T11:10:16Z`
  - Files: `backend/internal/httpapi/server.go`, `backend/migrations/versioned/checksums.sha256`, `web/app/globals.css`, `tasks/todo.md`, `tasks/plan.md`
  - Verify: `cd backend; go test ./...` and `cd web; npm run build`
  - State: `done`
  - Completed (UTC): `2026-07-27T11:10:16Z`
  - Result: `go test ./...` and `npm run build` passed; migration version 11 clean; desktop/mobile screenshots captured.
  - Changed: `backend/internal/settings/`, `backend/internal/storage/settings.go`, `backend/internal/httpapi/settings.go`, `backend/internal/httpapi/settings_test.go`, `backend/migrations/versioned/010_settings.*`, `web/app/settings/page.tsx`, `web/lib/settings.ts`, plus owned hunks in shared files.
  - Handoff: commit blocked because `backend/internal/httpapi/server.go` is partly staged by performance work and shared CSS/migration manifest contain unrelated active changes. Do not stage or commit them together.

## Supervisor Audit 2026-07-27

- [ ] Blocked: establish ownership and verification for all active changes.
  - Owner: `supervisor-agent`
  - Started (UTC): `2026-07-27T00:00:00Z`
  - Files: `tasks/todo.md`
  - Verify: `git diff --check`
  - State: `blocked`
  - Reason: 1,606 active changed lines have no task owner or lock; staged `docs/API.md` overlaps active API work.
- [ ] Required: wire loaded signal settings into engine threshold and score configuration.
  - Files: `backend/cmd/server/main.go`, `backend/internal/signals/engine.go`
  - Verify: `cd backend; go test ./...`
  - Reason: current startup loads settings but signal engine does not use them.
- [ ] Required: map compare `1d` timeframe to persisted `24h` outcome horizon.
  - Files: `backend/internal/httpapi/compare.go`, `backend/internal/httpapi/compare_test.go`
  - Verify: `cd backend; go test ./internal/httpapi`
- [ ] Required: prevent blocked audit candidates from consuming confirmation burst quota.
  - Files: `backend/internal/signals/engine.go`, `backend/internal/signals/engine_test.go`
  - Verify: `cd backend; go test ./internal/signals`

- [ ] Performance Proof-of-Edge dashboard
  - Owner: `opencode-performance`
  - Started (UTC): `2026-07-27T00:00:00Z`
  - Files: `backend/internal/performance/performance.go`, `backend/internal/performance/performance_test.go`, `web/components/performance-dashboard.tsx`, `docs/API.md`
  - Verify: `docker run --rm -v "${PWD}:/src" -w /src/backend golang:1.23 go test ./internal/performance ./internal/storage ./internal/httpapi`
  - State: `done`
  - Completed (UTC): `2026-07-27T00:00:00Z`
  - Result: backend focused tests and web build passed; committed `479a4ea`.

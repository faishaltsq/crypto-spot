# Dynamic Signal Threshold Tasks

## SELL Signal Engine 2026-07-28

- [ ] Build SPOT-only SELL signal system (tradeflow/structure/spoof features, SELL engine, dynamic threshold, invalidation, lifecycle, outcome, API, WebSocket, Terminal UI, Pair Diagnostic, Performance, notifications, tests).
  - Owner: `opencode-sell-engine`
  - Started (UTC): `2026-07-28T00:00:00Z`
  - Files: new packages under `backend/internal/features/tradeflow`, `backend/internal/features/structure`, `backend/internal/features/spoofing`, `backend/internal/signals/sell`; shared-file edits (claimed, small hunks only) in `backend/internal/config/config.go`, `backend/internal/domain/types.go`, `backend/internal/httpapi/server.go`, `backend/internal/realtime/hub.go`, `backend/cmd/server/main.go`, plus new migration `012_sell_signals` and web terminal/diagnostic/performance components.
  - Verify: `cd backend; go build ./...; go test ./...` and `cd web; npm run build`
  - State: `in_progress`
  - Note: does not touch `web/components/pairs/VirtualPairList.tsx`, `web/lib/settings.ts`, `backend/internal/httpapi/settings.go` beyond additive route registration; those active hunks from other agents are left untouched.

## Settings "unavailable" Build-Break Fix 2026-07-28

- [x] Remove unused `uuid` import breaking backend build
  - Owner: `opencode-settings-diagnosis`
  - Started (UTC): `2026-07-28T00:00:00Z`
  - Files: `backend/internal/httpapi/server.go`, `tasks/todo.md`
  - Verify: `cd backend; go build ./...; go test ./internal/httpapi/...`
  - State: `done`
  - Completed (UTC): `2026-07-28T00:00:00Z`
  - Result: `server.go` had an unused `github.com/google/uuid` import (WIP leftover from `opencode-sell-engine`'s in-progress edits), causing `go build` to fail entirely. Since backend didn't compile, every API route including `/api/v1/settings/system` was unreachable, which is why the Settings page showed "System status unavailable." Removed the one unused import line only; did not touch any sell-engine logic in this file. `go build ./...` and `go test ./internal/httpapi/...` now pass.
  - Handoff: `opencode-sell-engine` still owns `server.go` per its lock; if it re-adds `uuid` usage later, no conflict expected since this change only removed a dead import line.

## Docker Backend/Migrate Stale Image + Missing Checksum Fix 2026-07-28

- [x] Rebuild backend/migrate images from latest source and register migration 012 checksums
  - Owner: `opencode-settings-diagnosis`
  - Started (UTC): `2026-07-28T05:00:00Z`
  - Files: `backend/migrations/versioned/checksums.sha256`, `tasks/todo.md`
  - Verify: `docker compose build backend migrate; docker compose run --rm migrate up; docker compose up -d backend; cd backend; go test ./...`
  - State: `done`
  - Completed (UTC): `2026-07-28T05:45:00Z`
  - Result: Two stacked issues caused runtime errors after the CORS fix. (1) The running `backend` Docker image was built before the CORS header fix (`X-User-ID`/`X-User-Role` added to `AllowedHeaders` in commit `c33e239`), so preflight OPTIONS requests were silently rejected with no `Access-Control-Allow-Origin` header, causing browser CORS errors on `/settings/preferences`, `/settings/system`, `/admin/settings*`. (2) `checksums.sha256` was missing entries for `012_sell_signals.up.sql`/`.down.sql`, so `internal/migration.validateChecksums` caused the migrate container to treat the DB as already up to date at schema version 11 (never applying migration 012). This left `sell_score`/`sell_rule_score`/etc. columns missing on `signals`, causing `/api/v1/sell/signals` to return 500 `column "sell_score" does not exist`, which surfaced in the frontend as `Request failed: 500` from `initializeSignals` in `stores/market.ts`. Fixed by: rebuilding `backend` and `migrate` images from current source, adding the two missing checksum entries, running `migrate up` (now at schema version 12), and recreating the `backend` container. Verified `/api/v1/settings/system`, `/api/v1/sell/signals`, and `/api/v1/signals` all return 200, and `cd backend; go test ./...` passes.
  - Handoff: any agent that adds a new migration file must also add its checksum entries to `backend/migrations/versioned/checksums.sha256` in the same change, and rebuild+redeploy the `backend`/`migrate` Docker images (stale prebuilt images silently mask both build fixes and new migrations).

## Sell Signals Null List Crash Fix 2026-07-28

- [x] Return empty array instead of null from ListSellSignals; guard store against null API results
  - Owner: `opencode-settings-diagnosis`
  - Started (UTC): `2026-07-28T05:50:00Z`
  - Files: `backend/internal/storage/sell.go`, `web/stores/market.ts`, `tasks/todo.md`
  - Verify: `cd backend; go build ./...; go test ./...` and `docker compose build backend; docker compose up -d backend`
  - Result: `signals` table was freshly empty right after migration 012 applied (no SELL signals recorded yet). `Repository.ListSellSignals` declared `var out []SellSignalDetail`, which stays Go `nil` when zero rows match and serializes to JSON `null`. `stores/market.ts:initializeSignals` stored that `null` directly into `sellSignals`, and `PairDiagnostic.tsx:19` called `.find()` on it, throwing `Cannot read properties of null (reading 'find')` and crashing the terminal page. Fixed by initializing `out := []SellSignalDetail{}` so empty results serialize to `[]`, and added a `?? []` guard in `initializeSignals` as defense-in-depth against any other endpoint returning null. Verified `GET /api/v1/sell/signals` now returns `{"signals":[]}`, `go build`/`go test ./...` pass, and rebuilt+redeployed the `backend` image.
  - State: `done`
  - Completed (UTC): `2026-07-28T06:00:00Z`

## Terminal Data Quality Missing Metrics Fix 2026-07-27

- [x] Render incomplete quality metrics safely
  - Owner: `opencode-terminal-quality-fix`
  - Started (UTC): `2026-07-27T14:05:00Z`
  - Files: `web/components/diagnostics/PairDiagnostic.tsx`, `web/tests/e2e.spec.ts`, `tasks/todo.md`
  - Verify: `cd web; npx playwright test tests/e2e.spec.ts --project=desktop --grep "terminal data quality"; npm run build`
  - State: `done`
  - Completed (UTC): `2026-07-27T14:05:00Z`
  - Result: terminal Data Quality accepts backend snake_case fields and renders absent numeric values as `Unavailable`.
  - Verify: focused desktop Playwright regression and `cd web; npm run build` passed.

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

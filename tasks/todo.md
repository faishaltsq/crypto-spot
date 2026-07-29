# Dynamic Signal Threshold Tasks

## Refactor BUY Engine Configuration (Fase 2) 2026-07-29

- [x] Replace hardcoded BUY engine thresholds with a real EngineConfig struct wired from env/Admin Settings to runtime decisions; remove dead config fields.
  - Owner: `opencode-buy-engine-config`
  - Started (UTC): `2026-07-29T18:00:00Z`
  - Files: `backend/internal/signals/config.go`, `backend/internal/signals/engine.go`, `backend/internal/signals/engine_test.go`, `backend/cmd/server/main.go`, `backend/internal/config/config.go`
  - Verify: `cd backend; go build ./...; go test ./...`
  - State: `done`
  - Result: Rewrote `signals.EngineConfig` to hold only fields BUY actually consumes: `ConfirmScore`, `MaxSpoofScore`, `MinTrendAlignment`, `PairCooldown`, `MaxNewPerMinute`, plus 3 config-only `MaxActive*` limits (parity with SELL, not yet enforced on hot path by either engine). Deleted dead consts `MinRuleScoreForSetup`, `MinRuleScoreForConfirmed`, `MinDataQualityForSignal`, `MaxSpoofScoreForConfirmed`, `MinTrendAlignmentForConfirmed`, `MaxActiveSignalsGlobal`, and the never-read `Engine.activeCount`/`minScore`/`confirmScore` fields. `engine.go` now reads all gates from a snapshot of `e.cfg` taken under lock in `Evaluate`. Core bug fixed: `New()` now takes `EngineConfig` via new `signals.FromAppConfig(&cfg)`, so `SignalConfirmScore`/`SignalMaxSpoofScore`/`SignalMaxNewPerMinute` (previously loaded, validated, then ignored while engine used hardcoded 80/60/5) now reach the engine at startup, after `applyStoredSettings`. Added config-driven `SignalMinTrendAlignment` (env `SIGNAL_MIN_TREND_ALIGNMENT`, default 0.20; was hardcoded). Added `Engine.SetConfig()` live-reload method (validates before swap) mirroring `SetThresholdConfig`. Decisions (user-approved): `MinModelProbability` deleted (no real model_probability exists); score/data-quality gate fields NOT added to BUY (threshold-based arch has no consumption point); active-count enforcement deferred to a future BUY+SELL-symmetric fase. 4 new integration tests prove config flips CONFIRMED/SETUP and that SetConfig hot-swaps + rejects invalid. `go build ./...` and `go test ./...` both green. Runtime (post-boot) Admin Settings hot-reload deferred: needs `httpapi/server.go` (out of lock) to thread the engine into `saveAdminSettings` — see follow-up below.
  - Follow-up: (1) Wire runtime Admin Settings PUT -> `signalEngine.SetConfig` (thread engine through `httpapi.New`). (2) Design real `MaxActiveGlobal/PerPair/PerCluster` enforcement for BOTH BUY and SELL (needs an active-signal count query; tier lives in `feature_snapshot` JSONB, not a `signals` column). (3) SELL engine has the same dead-field pattern (`MaxActive*`, `activeCount`, `RequireExecutedTradeConfirm`, `RequireFailedReclaim`, `MinModelProbability`, `MinTradeflowScore`) — out of this fase's scope. (4) Stale comment referencing removed `MinDataQualityForSignal` const in `backend/internal/config/sell_load.go:19`.

## Refactor Signals Record Kind 2026-07-29

- [x] Implement record_kind states and new fields to differentiate actionable signals from candidates.
  - Owner: `opencode-refactor-signals`
  - Started (UTC): `2026-07-29T17:00:00Z`
  - Files: `backend/internal/domain/types.go`, `backend/internal/domain/signal_status.go`, `backend/internal/signals/engine.go`, `backend/internal/signals/sell/engine.go`, `backend/internal/signals/sell/protective_sell.go`, `backend/internal/signals/sell/engine_test.go`, `backend/migrations/versioned/013_signal_record_kind.up.sql`, `backend/migrations/versioned/checksums.sha256`
  - Verify: `cd backend; go build ./...; go test ./...` and `cd web; npm run build`
  - State: `done`
  - Result: Fixed `WATCH` status handling in BUY engine to avoid premature drops. Refactored SELL engine (`backend/internal/signals/sell/engine.go`) to emit `BLOCKED` audit signals instead of silently returning `nil, false` when `hardGates` fail. `protective_sell.go` now sets `RecordKind`, `DecisionStage`, `BlockedStage` when producing `baseSignal`. SELL engine tests were updated to expect a returned signal with `Status = "BLOCKED"` instead of `nil`. Updated migration `013_signal_record_kind.up.sql` to also mark existing `status = 'BLOCKED'` as `record_kind = 'BLOCKED_AUDIT'` correctly. Regenerated hash in `checksums.sha256`. All backend tests passed!


## Fix sparse/broken candle chart for newly-added pairs 2026-07-29

- [x] Backfill REST klines for pairs added to the universe after startup; show placeholder instead of a stretched single candle.
  - Owner: `opencode-chart-backfill`
  - Started (UTC): `2026-07-29T00:00:00Z`
  - Files: `backend/internal/exchange/gate/connection_manager.go`, `web/components/chart/LightweightMarketChart.tsx`, `web/app/globals.css`
  - Verify: `cd backend; go build ./...; go test ./...` and `cd web; npm run build`
  - State: `done`
  - Result: root cause was `ConnectionManager.UpdatePairs` only firing `HistoryFetcher.Backfill` once, when `cm.active` was empty (startup). Pairs added on a later universe refresh (e.g. `PairUniverseRefreshMin` ticker, low-liquidity/newly-listed pairs like EVAA/USDT) never got the 300-bar REST kline backfill and only accumulated candles from live WS ticks going forward, leaving their chart with 1-2 candles. Added `diffNewPairs` to backfill any symbol not already in `cm.active` on every `UpdatePairs` call, not just the first. Frontend: `LightweightMarketChart.tsx` now shows a "Collecting candle data" placeholder when `formattedData.length < 5` instead of letting lightweight-charts' `fitContent()` stretch a near-empty series into one giant bar. Added `@keyframes spin` to `globals.css` for the placeholder spinner. Backend `go build`/`go test ./...` passed (all packages ok, `gate` has no test files pre-existing). `cd web; npm run build` passed.

## Hide sub-threshold signals from UI to reduce lag 2026-07-29

- [x] Filter below-minimum-score signals out of history/dashboard rendering (still scanned/stored in background).
  - Owner: `opencode-ui-perf`
  - Started (UTC): `2026-07-29T00:00:00Z`
  - Files: `web/lib/api.ts`, `web/app/signals/page.tsx`, `web/components/dashboard.tsx`
  - Verify: `cd web; npm run build`
  - State: `done`
  - Result: fixed `getSignalsFiltered`/`exportSignalsCSV` query param mismatch (`score_min`/`created_from` -> `scoreMin`/`createdFrom` to match backend `parseSignalFilter`). `/signals` page now defaults `scoreMin` filter to `cfg.signalMinScore` from `GET /api/v1/config` (user can still lower it manually). Dashboard realtime `signal.created`/`sell.signal.created` handler now skips appending signals below `signalMinScore` to the live history panel. Follow-up: Terminal page right-side `RightSignalPanel.tsx` History tab (`web/components/signals/RightSignalPanel.tsx:73-78`) also had zero score filtering (just `base.slice(0,50)` in insertion order) — added `.filter(ruleScore >= signalMinScore)` + sort by `createdAt` desc, min score fetched from `GET /api/v1/config` (defaults to 70 until loaded). Backend unchanged: scanning/persistence of sub-threshold candidates is untouched, only UI rendering is filtered. `cd web; npm run build` passed (after clearing stale `.next` cache unrelated to this change).

## SL-Hit Stale Active Signal + Stale Docker Image Fix 2026-07-28

- [x] Fix SELL signals staying in Active Signals / chart drawings not clearing after stop-loss hit; fix stale Docker images masking source changes.
  - Owner: `opencode-sl-active-fix`
  - Started (UTC): `2026-07-28T16:00:00Z`
  - Files: `backend/internal/signals/sell/outcome.go`, `tasks/todo.md`
  - Verify: `cd backend; go build ./...; go test ./internal/signals/sell/...` and manual `GET /api/v1/signals/active` returns 200
  - State: `done`
  - Completed (UTC): `2026-07-28T16:20:00Z`
  - Result: `outcome.go`'s `evaluate()` already closed signals via `CloseSellSignal` when `EvaluateDirectional` returned `Invalidated`, but always tagged the reason `SUPPORT_RECLAIMED` even when the invalidation was actually a stop-loss breach (`DirectionalReturn > 0`, i.e. price moved against the protective SELL). Added a branch so SL breaches record `invalidation_reason = "STOP_LOSS_BREACHED"` instead of being mislabeled, while the actual close/broadcast path (status -> `INVALIDATED`, `signal.updated` WS broadcast) was unchanged and already correct. Separately, discovered the running `crypto-spot-signal-backend` Docker image was stale (built before the `/api/v1/signals/active` route and several `httpapi`/`storage` changes in the working tree), causing chi to route `GET /signals/active` into the old `/signals/{id}` handler and fail with `invalid input syntax for type uuid: "active"` (500). Rebuilt and recreated `backend`, `ai-service`, and `web` images/containers from current source; confirmed `/api/v1/signals/active` now returns 200 with `{"signals":[],...}` and ai-service reports `provider":"deepseek"` per the user's new `.env` key.
  - Verify: `go build ./...` and `go test ./internal/signals/sell/...` passed. `docker compose build backend ai-service web` succeeded (`npm run build`/`npx tsc --noEmit` clean). Live `GET /api/v1/signals/active` returns `200`.
  - Note: user's local `.env` (gitignored) now has `AI_ENABLED=true`, `AI_PROVIDER=deepseek`, `DEEPSEEK_API_KEY` set, `AI_MODEL=deepseek-chat` — no code change needed for that part, only container recreation since `env_file` is applied at container-create time, not live-reloaded.
  - Committed: `82f4890` on branch `fix/sell-signal-realtime-wiring`, pushed to `origin/fix/sell-signal-realtime-wiring`. This commit also included prior uncommitted worktree changes from the "Unified BUY/SELL Signal Lifecycle Contract" and "SELL Signal Logic Fix" tasks above (both already marked `done` in this file with no conflicting active lock at commit time).

## Unified BUY/SELL Signal Lifecycle Contract 2026-07-28

- [ ] Backend-owned `isActive`/`direction`/`strategy`/`lifecycleGroup` contract on every signal response + unified `/api/v1/signals/active` endpoint + realtime `signal.updated` lifecycle broadcasts; frontend unified signal store/selectors consuming that contract instead of re-deriving status client-side.
  - Owner: `opencode-signal-lifecycle-unify`
  - Started (UTC): `2026-07-28T06:30:00Z`
  - Files: `backend/internal/domain/types.go` (additive struct fields + `Enrich()` method only), `backend/internal/domain/signal_status.go` (new file), `backend/internal/domain/signal_status_test.go` (new file), `backend/internal/storage/postgres.go` (additive `Enrich()` call + `UpdateOutcome` return-value change), `backend/internal/storage/sell.go` (additive `ListActiveSignals` + `Enrich()` call), `backend/internal/signals/engine.go` (additive `Enrich()` call), `backend/internal/signals/sell/protective_sell.go`/`take_profit.go`/`outcome.go` (additive `Enrich()` calls + broadcast wiring), `backend/internal/httpapi/server.go` (additive route registration only, one line), `backend/internal/httpapi/sell.go` (additive handler), `backend/cmd/server/main.go` (wired previously-orphaned `outcomeLoop` goroutine + hub param), plus new `web/stores/signals.ts`, `web/lib/signal-status.ts`, `web/lib/signal-normalize.ts`.
  - Verify: `cd backend; go build ./...; go test ./...` and `cd web; npm run build`
  - State: `done`
  - Completed (UTC): `2026-07-28T07:30:00Z`
  - Result: Added `domain.Signal.Enrich()` populating `isActive`/`direction`/`strategy`/`lifecycleGroup` on every BUY/SELL signal row (called from `scanSignal`, `scanSellSignal`, `engine.go`, `sell/protective_sell.go`, `sell/take_profit.go`). Added `GET /api/v1/signals/active` (unified BUY+SELL, filterable by direction/strategy/symbol/timeframe). Wired the previously-orphaned `outcomeLoop` goroutine into `main()` and made both it and `sell.OutcomeEvaluator` broadcast `signal.updated` over the WebSocket hub whenever a signal transitions to a terminal state, so clients no longer have to poll to learn a signal left the active set. Frontend: added `lib/signal-status.ts` (single `isSignalActive`/`signalDirection`/`signalLifecycleGroup` helpers) and `lib/signal-normalize.ts` (defensive fallback for stale payloads), wired `signal.updated` in `GlobalRealtime.tsx`, removed the client-side price-based status-mutation logic from `stores/market.ts` (backend is now sole source of truth), and replaced all `status === 'ACTIVE'`/`.includes('BUY')` string-matching in `RightSignalPanel.tsx`, `PairDiagnostic.tsx`, `VirtualPairList.tsx`, `LightweightMarketChart.tsx`, and `app/signals/page.tsx` with the centralized helpers.
  - Verify: `cd backend; go build ./...` and `go test ./...` passed (added `internal/domain/signal_status_test.go`). `cd web; npx tsc --noEmit` and `npm run build` passed.
  - Handoff: `opencode-sell-engine`'s claimed files (`main.go`, `domain/types.go`, `server.go`, `realtime/hub.go`) had no active `.agent-locks` entry and no conflicting diff at start of this task; all edits here were strictly additive. If that task resumes, its SELL scoring/threshold logic is untouched — only new `Enrich()` calls and one new route line were added.

## SELL Signal Logic Fix (bearish signals never surface) 2026-07-28

- [ ] Fix SELL signal logic so PROTECTIVE_SELL surfaces in bearish markets: (1) priority switch falls back to PROTECTIVE_SELL when AVOID_ENTRY doesn't fire, (2) trend gate sensitive to low-TF bearish + strong flow when HTF hasn't crossed over, (3) RuleScore renormalizes weights when wall data absent so neutral 50 doesn't drag score, (4) lower over-strict config defaults toward BUY parity.
  - Owner: `opencode-sell-signal-logic-fix`
  - Started (UTC): `2026-07-28T13:02:09Z`
  - Files: `backend/internal/signals/sell/engine.go`, `backend/internal/signals/sell/protective_sell.go`, `backend/internal/signals/sell/score.go`, `backend/internal/config/sell_load.go`
  - Verify: `cd backend; go build ./...; go test ./internal/signals/sell/...`
  - State: `done`
  - Completed (UTC): `2026-07-28T13:10:00Z`
  - Note: `opencode-sell-engine` claim above has NO active `.agent-locks/*.lock` entry and no heartbeat; per AI_COORDINATION.md `.agent-locks/*.lock` is source of truth for file ownership. Supervisor (user) directed this fix. Only the four listed SELL files touched.
  - Result: (1) `engine.go` — priority switch no longer swallows SELL evaluation: when `HasCandidateSignal` and AVOID_ENTRY gates don't trip, and when `HasActivePosition` but neither EXIT_WARNING nor TAKE_PROFIT fires, both now fall back to `evaluateProtectiveSell` so bearish pairs still surface instead of vanishing behind a low-bar BUY_SETUP. (2) `protective_sell.go` — extracted `bearishTrendConfirmed`: primary weighted-alignment gate kept, plus a low-timeframe override (5m/15m bearish + AggressiveSellRatio>=0.60 + negative CVD slope + not counter-trended) so fresh breakdowns surface before HTF EMA crossovers catch up. (3) `score.go` — RuleScore renormalizes trend/flow/structure weights when NO bid-wall event is observed, instead of feeding a hardcoded neutral 50 into the 15% wall slice that dragged totals below SetupScore. (4) `sell_load.go` — `SELL_MIN_DATA_QUALITY` 80->75 (BUY parity), `SELL_MIN_TIMEFRAME_ALIGNMENT` 60->40.
  - Verify: `go build ./...` and full `go test ./...` passed.
  - Tests added (`engine_test.go`): `TestProtectiveSellFallbackWhenBuyCandidateButNoAvoidEntry`, `TestBearishTrendConfirmedLowTFOverride`, `TestRuleScoreRenormalizesWhenNoWallEvent` — regression coverage for all three logic changes.
  - Not changed (deliberate): SampleStatus hard gate (`SELL_MIN_TRADE_COUNT`/`NOTIONAL`) left as-is — it is a genuine liquidity precondition; lowering it risks false SELL signals on illiquid altcoins. Flag for supervisor if low-liquidity SELL coverage is desired.

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

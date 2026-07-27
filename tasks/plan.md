# Implementation Plan: Dynamic Signal Threshold

## Overview
Make dynamic threshold a decision gate, audit every calculation, expose it through existing signal API, and render its full breakdown in Signal Evidence.

## Architecture Decisions
- `backend/internal/signals/threshold` owns threshold config, classifications, reason codes, validation, and calculation.
- Existing `signals.threshold_detail` JSONB stores full breakdown; dedicated generated-column migration is unnecessary because JSONB is already persisted and returned by all signal queries.
- `STRONG_DOWNTREND`, invalid data quality, high spoof risk, and low liquidity block confirmation. Blocked candidates are persisted as `BLOCKED` signal audits so UI/API can show reasons.
- Feature snapshot carries explicit regime, volatility percentile, and correlation state. Missing regime/volatility are conservative and auditable.

## Task List

### Phase 1: Threshold Foundation
- [ ] Task 1: Add threshold models, defaults, validation, adjustments, calculator, reason codes.
  - Acceptance: configurable adjustments produce final threshold; hard blocks are explicit.
  - Verify: `go test ./internal/signals/threshold`.
- [ ] Task 2: Extend feature data with threshold inputs and calculate deterministic defaults.
  - Acceptance: every evaluated feature has regime, volatility, correlation values or explicit missing state.
  - Verify: `go test ./internal/features`.

### Checkpoint: Foundation
- [ ] Threshold unit tests pass.

### Phase 2: Signal Integration
- [ ] Task 3: Use calculation before confirmation decision; persist blocked candidates and full audit breakdown.
  - Acceptance: confirmation compares `RuleScore` against `FinalThreshold`; hard blocks cannot become `BUY_CONFIRMED`.
  - Verify: `go test ./internal/signals ./internal/storage`.
- [ ] Task 4: Add migration audit index/version and configuration wiring.
  - Acceptance: schema migration validates and application loads configurable settings.
  - Verify: `go test ./internal/migration ./cmd/migrate`.

### Checkpoint: Backend
- [ ] `go test ./...` passes.

### Phase 3: UI Evidence
- [ ] Task 5: Type and render every threshold audit field in Signal Evidence.
  - Acceptance: base, each adjustment, final, score, pass/fail, blocks, reasons, version shown.
  - Verify: `npm run build` and frontend test.

### Checkpoint: Complete
- [ ] API response includes persisted breakdown.
- [ ] Full backend and frontend verification passes.

## Risks and Mitigations
| Risk | Impact | Mitigation |
|---|---|---|
| Missing market inputs | Incorrect threshold | Conservative `UNDETERMINED` and `MISSING_*` reason codes |
| Existing migrations checksum-gated | Startup failure | Update manifest only with migration hashes |
| Blocked candidates previously discarded | No audit/UI evidence | Persist `BLOCKED` candidate signals |

## Open Questions
- None. Defaults: `STRONG_DOWNTREND` blocks; missing volatility adds `0` with reason code.

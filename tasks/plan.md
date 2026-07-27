# Implementation Plan: Paper Execution Simulation Integration

## Objective
Persist per-notional paper fills for confirmed signals, evaluate exit fills from live order book depth, then expose net outcome returns to performance APIs. No order execution, private exchange API, or trading UI is in scope.

## Design
- Version 7 adds `paper_execution_simulations`, unique on `(signal_id, notional)`.
- Entry uses ask-side order-book walk. Exit uses bid-side walk. Missing book is `INCOMPLETE`; depth shortfall is `PARTIAL_FILL`.
- Monetary values are USDT. Return decimals and percentages are separate API fields.
- Outcome horizon values carry gross/net returns. Performance aggregates persisted net 1h horizon values, excluding unavailable simulation.

## Tasks
- [ ] Add migration and simulation math tests.
- [ ] Persist entry simulations immediately after confirmed signal save.
- [ ] Update exits through outcome loop; preserve per-notional entry state.
- [ ] Add simulations to signal API/history/export and net performance summary.
- [ ] Verify migration, integration, build, review, commit.

## Risks
| Risk | Mitigation |
| --- | --- |
| Missing levels | Persist explicit incomplete status; do not create zero costs. |
| Insufficient levels | Persist fill/unfilled amount and partial status. |
| Duplicate scanner run | DB unique constraint and upsert protect entry records. |
| Stale outcome | Horizon JSON is updated from persisted simulation result, not raw price-only return. |

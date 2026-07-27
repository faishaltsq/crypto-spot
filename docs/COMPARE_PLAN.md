# Implementation Plan: Compare SPOT Pairs

## Dependencies

`market.Store` + runtime feature state + universe + quality service + repository outcomes
-> compare snapshot API
-> web contract/client
-> Compare route
-> global WebSocket pair subscription

## Tasks

1. Backend contract and snapshot service.
   - Acceptance: validated 2-4 pair request returns one timestamp, active/inactive status, live metrics, normalized candles, historical aggregates.
   - Verify: focused `go test ./internal/httpapi`.

2. Backend cache and realtime filtering.
   - Acceptance: equivalent request is cached briefly; pair unsubscribe does not close WebSocket or affect other subscriptions.
   - Verify: focused Go tests.

3. Frontend route, URL state, one-request client.
   - Acceptance: selector preserves valid URL state, supports search/remove, blocks duplicate and over-limit selection, renders all required groups.
   - Verify: `npm run build`.

4. Responsive and browser verification.
   - Acceptance: desktop side-by-side summary/chart; tablet stack; mobile controlled metric view; clean browser request path.
   - Verify: Playwright and screenshots.

5. Full quality gate and commit.
   - Acceptance: tests/build pass, review findings resolved, no unrelated files staged.
   - Verify: repository test commands and staged diff inspection.

## Risks

| Risk | Mitigation |
| --- | --- |
| Existing global WebSocket has no subscriptions | Add optional `compare.snapshot` subscriptions only; preserve existing global events. |
| Candle coverage shorter than lookback | Return available points and data-quality metadata; do not synthesize history. |
| Outcome schema stores horizon JSON | Aggregate only evaluated horizon result matched to requested timeframe; return insufficient sample otherwise. |
| No Watchlist data source | Render disabled control with phase explanation. |

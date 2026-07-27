# Spec: Compare SPOT Pairs

## Objective

Route `/compare` compares two to four active SPOT pairs over one timeframe and lookback. The page obtains every selected-pair metric from one backend snapshot request and receives filtered live feature updates through the existing global WebSocket connection. It does not implement Watchlist or Settings.

## Commands

```text
Backend test: cd backend && go test ./...
Backend build: cd backend && go build ./cmd/server
Web build: cd web && npm run build
Browser test: cd web && npx playwright test
```

## Contract

```http
GET /api/v1/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h
```

`pairs` contains 2-4 unique `BASE_USDT` symbols. `timeframe` is one of `1m`, `5m`, `15m`, `30m`, `1h`, `4h`, `8h`, `1d`. `lookback` is one of `1h`, `4h`, `24h`, `7d`, `30d`.

The response contains one `snapshotAt` timestamp, request filters, available pair metrics, and unavailable-pair records. Metric values absent from source data serialize as `null`; the backend never manufactures metrics. Historical results aggregate evaluated signal outcomes for each pair in requested lookback. Quote values are USDT. Price series are percent-normalized from each pair's first valid close.

## Architecture

- `backend/internal/httpapi/compare.go` validates query input, reads active universe, market snapshot, feature state, quality report, and historical repository aggregate.
- Short in-process cache keys complete validated query state for three seconds. Cached responses retain their original `snapshotAt` timestamp.
- `realtime.Hub` supports optional pair subscriptions for `compare.snapshot`; existing events remain global for backward compatibility.
- `/compare` owns URL state, makes exactly one REST snapshot request, and registers selected pairs with global realtime state.
- Watchlist-only is visibly unavailable in this phase because Watchlist has no persisted source. No Watchlist behavior is implemented.

## Testing Strategy

- Go unit tests: query validation, limits, duplicate/inactive/invalid pairs, normalization, null metrics, cache, timestamp consistency, historical insufficient samples, subscription cleanup.
- Playwright: URL persistence, refresh, one compare request, pair removal, mobile layout.
- Build: Go backend plus Next.js production build.

## Boundaries

- Always: validate query parameters, parameterize historical aggregate, preserve `null` metrics, keep REST to one snapshot call.
- Ask first: database schema changes, Watchlist persistence, dependencies, changed global event behavior.
- Never: static market data, fake scores, base-volume comparison, Watchlist or Settings implementation, order execution.

## Success Criteria

- Compare page renders backend snapshot for 2-4 active pairs.
- URL is source of truth for pairs, timeframe, and lookback.
- Page exposes required comparison groups, evidence, warnings, and freshness.
- One global WebSocket connects once and filters Compare updates by selected symbols.
- Validation, cache, unavailable, stale, partial, and insufficient-sample paths are covered.

# HTTP and WebSocket API

Base backend address: `http://localhost:8080`

## REST

### Health

```http
GET /health
```

### Scanner

```http
GET /api/v1/scanner
```

Returns the current feature snapshot for configured pairs, sorted by rule score.

### Signals

```http
GET /api/v1/signals?limit=100
GET /api/v1/signals/{id}
```

### Pair state

```http
GET /api/v1/pairs/BTC_USDT
```

Returns in-memory candles, top order book levels, book metrics, and the latest feature snapshot.

### Performance

```http
GET /api/v1/performance/summary
```

Returns evaluated signal counts, target hit rate, average returns, MFE, and MAE.

### Compare

```http
GET /api/v1/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h
```

Accepts two to four unique active `_USDT` pairs. `timeframe`: `1m`, `5m`, `15m`, `30m`, `1h`, `4h`, `8h`, `1d`. `lookback`: `1h`, `4h`, `24h`, `7d`, `30d`.

Response contains one consistent `snapshotAt`, live pair metrics, normalized price-performance series, historical sample metadata, freshness/data-quality metadata, and unavailable-pair records. Endpoint uses a three-second server cache. Missing source metrics remain `null`.

### Proof-of-Edge performance

```http
GET /api/v1/performance?dateFrom=2026-07-01T00:00:00Z&dateTo=2026-07-31T23:59:59Z&pair=BTC_USDT&notional=100
```

Optional filters: `pair`, `tier`, `timeframe`, `signalStatus`, `scoreBucket`,
`marketRegime`, `ruleVersion`, `modelVersion`, `aiDecision`, and `notional`.

Returns only execution-simulation performance. Return unit is decimal: `0.01 = 1%`.
`INCOMPLETE_SIMULATION` is excluded from return metrics rather than counted as zero.
Response includes metric definitions, units, sample counts, filters, calculation time,
simulation-status counts, reliability status, Edge Score components, breakdowns, and charts.

### Public configuration

```http
GET /api/v1/config
```

The response always reports `executionEnabled: false`.

## WebSocket

```text
ws://localhost:8080/ws
```

Events:

```json
{
  "event": "scanner.snapshot",
  "timestamp": "2026-07-26T10:00:00Z",
  "data": []
}
```

Compare clients may send one message on global connection:

```json
{"channel":"compare","action":"subscribe","pairs":["BTC_USDT","ETH_USDT"]}
```

`unsubscribe` removes only specified pairs. Compare updates use `compare.snapshot` and are emitted only to subscribed clients.

```json
{
  "event": "signal.created",
  "timestamp": "2026-07-26T10:00:00Z",
  "data": {
    "symbol": "BTC_USDT",
    "type": "BUY_SETUP"
  }
}
```

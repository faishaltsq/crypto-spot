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

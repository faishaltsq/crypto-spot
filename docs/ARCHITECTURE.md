# Architecture

## Data flow

```text
Gate SPOT WebSocket and REST
        |
        v
Go market ingestor
  - trades
  - book ticker
  - incremental order book
  - candles
        |
        v
In-memory pair state
        |
        +--> PostgreSQL/TimescaleDB
        |      - candles
        |      - orderbook metrics
        |      - feature snapshots
        |      - signals and outcomes
        |
        +--> Feature engine
                  |
                  v
             Rule score
                  |
                  v
          AI review service
                  |
                  v
             Signal engine
                  |
        +---------+---------+
        |                   |
        v                   v
 REST and WebSocket      Redis cache
        |
        v
 Next.js dashboard and browser notification
```

## Service boundaries

### Go backend

The backend owns exchange connectivity, local order book state, feature calculation, signal lifecycle, persistence, REST endpoints, and the dashboard WebSocket. It never places an order.

### Python AI service

The service receives a compact feature snapshot. It does not receive raw order book deltas. It returns `CONFIRM`, `WAIT`, or `REJECT` through a strict response model. A deterministic provider remains available when external AI is disabled or unavailable.

### Next.js web

The dashboard renders scanner values, multi-timeframe trend states, candlesticks, order book depth, signal history, and browser notifications. The browser must be open or running the installed site for the included notification flow. Fully closed-browser push requires a VAPID push subscription service and is intentionally outside this paper-signal MVP.

## Failure rules

A candidate cannot become a signal when data quality is below the threshold. Order book sequence gaps mark the book unsynchronized and trigger a REST snapshot refresh. AI provider failure falls back to deterministic review. Database failure prevents a candidate from being broadcast as a stored signal.

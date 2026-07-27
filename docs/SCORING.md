# Scoring model

The included score is an initial heuristic. It is not a trained profitability model.

## Base score

```text
trend score        30%
volume score       22%
order flow score   23%
liquidity score    15%
data quality       10%
```

The engine subtracts spoofing and overextension penalties. Every component is bounded from 0 to 100.

## Data quality

The engine penalizes:

- Unsynchronized order book
- Market data older than 30 seconds
- Insufficient one-minute candle history
- Low recent trade count

A score cannot create a signal when data quality is below 60.

## Spoofing heuristic

The current implementation records removed quote liquidity and compares it with recent traded quote volume. It also penalizes shallow books. This detects suspicious cancellation pressure, but it cannot prove intent. Calibrate it per pair using recorded order book metrics.

## Status thresholds

```text
0 to 59.99    NO_SIGNAL
60 to 69.99   WATCH
70 to 79.99   BUY_SETUP
80 to 100     BUY_CONFIRMED_CANDIDATE
```

The signal engine applies a second threshold, cooldown, AI review, and data-quality filter before storing a signal.

## Validation requirements

Before treating the score as useful, measure outcomes by pair, timeframe, and market regime. Include trading fees and estimated slippage. Use chronological walk-forward validation. Do not use random train-test splitting for market time series.

# Known limitations

1. The scoring model is heuristic. It has not been trained or calibrated against the user's own historical dataset.
2. The spoof score cannot prove spoofing intent. It measures suspicious removed liquidity relative to traded volume.
3. Pair selection is configured through `GATE_PAIRS`. The package does not automatically promote and demote every Gate pair by liquidity tier.
4. Raw trades are held in a rolling in-memory window. Aggregated candle, order book metric, feature, signal, and outcome data are persisted.
5. Browser notifications work while the site is open or running. Fully closed-browser push is not included.
6. The initial REST candle backfill is limited to 120 rows per pair and timeframe.
7. Signal return evaluation samples the current market price on a schedule. It is not a tick-perfect execution simulator.
8. Fee and slippage are not deducted from outcome returns in this MVP.
9. DeepSeek and Grok compatibility depends on a model that supports the configured response mode.
10. There is no authenticated exchange client, portfolio state, position sizing, or automatic order execution.

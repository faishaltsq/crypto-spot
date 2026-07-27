import { test, expect } from '@playwright/test';

const snapshot = {
  snapshotAt: '2026-07-27T10:00:00Z', timeframe: '15m', lookback: '24h', cacheTtlSeconds: 3,
  filters: { normalizePerformance: true, activeSignalOnly: false, watchlistOnlyAvailable: false }, unavailable: [],
  pairs: ['BTC_USDT', 'ETH_USDT'].map((symbol, index) => ({
    symbol, rank: index + 1, tier: 1, price: index ? 3500 : 120000, change24hPercent: index ? -1.2 : 2.4, quoteVolume24h: 1000000,
    relativeVolume: 1.2, spreadBps: 4, bidDepthQuote: 200000, askDepthQuote: 190000, liquidityScore: 85,
    estimatedSlippage100: 5, estimatedSlippage500: 25, cvd: 10000, orderbookImbalance: .2, spoofScore: 8,
    trend: 'bullish', momentum: 1.1, atrPercent: 2, multiTimeframeAlignment: 80, signalScore: 78, dynamicThreshold: 70,
    dataQualityScore: 95, dataQualityStatus: 'VALID', activeSignal: true,
    historical: { netReturn: 1.5, winRate: .6, sampleCount: 24, netExpectancy: .4, mfe: 2, mae: -1, insufficientSample: false },
    pricePerformance: [{ time: '2026-07-27T09:00:00Z', value: 0 }, { time: '2026-07-27T10:00:00Z', value: 2 }],
    supportingEvidence: ['POSITIVE_CVD', 'HIGH_SPOOF_RISK', 'HIGH_SPOOF_RISK'], contradictingEvidence: [], freshness: { lastMarketUpdate: '2026-07-27T10:00:00Z', isStale: false, bookSynced: true }, partialMetrics: [],
  })),
};

test('compare keeps URL selection and requests one snapshot endpoint', async ({ page }) => {
  await page.route('**/api/v1/market/universe/', route => route.fulfill({ json: [{ Symbol: 'BTC_USDT', Tier: 1 }, { Symbol: 'ETH_USDT', Tier: 1 }, { Symbol: 'SOL_USDT', Tier: 2 }] }));
  let compareRequests = 0;
  await page.route('**/api/v1/compare?*', route => { compareRequests += 1; return route.fulfill({ json: snapshot }); });
  await page.goto('/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h');
  await expect(page.getByRole('heading', { name: 'Pair comparison' })).toBeVisible();
  await expect(page.getByText('Summary matrix')).toBeVisible();
  await expect(page.getByText('Normalized price performance')).toBeVisible();
  await expect(page).toHaveURL(/pairs=BTC_USDT%2CETH_USDT/);
  await page.getByLabel('Remove ETH_USDT').click();
  await expect(page).toHaveURL(/pairs=BTC_USDT/);
  expect(compareRequests).toBe(1);
});

test('compare fits responsive viewport', async ({ page }, testInfo) => {
  await page.route('**/api/v1/market/universe/', route => route.fulfill({ json: [] }));
  await page.route('**/api/v1/compare?*', route => route.fulfill({ json: snapshot }));
  await page.goto('/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h');
  await expect(page.getByRole('heading', { name: 'Pair comparison' })).toBeVisible();
  await page.screenshot({ path: `test-results/compare-${testInfo.project.name}.png`, fullPage: true });
});

test('compare renders repeated evidence without key warnings', async ({ page }) => {
  const duplicateKeyWarnings: string[] = [];
  page.on('console', message => {
    if (message.type() === 'error' && message.text().includes('same key')) duplicateKeyWarnings.push(message.text());
  });
  await page.route('**/api/v1/market/universe/', route => route.fulfill({ json: [] }));
  await page.route('**/api/v1/compare?*', route => route.fulfill({ json: snapshot }));
  await page.goto('/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h');
  await expect(page.getByText('HIGH_SPOOF_RISK', { exact: true })).toHaveCount(4);
  expect(duplicateKeyWarnings).toEqual([]);
});

test('terminal data quality renders missing metrics safely', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/v1/scanner', route => route.fulfill({ json: [] }));
  await page.route('**/api/v1/signals?*', route => route.fulfill({ json: [] }));
  await page.route('**/api/v1/terminal/BTC_USDT', route => route.fulfill({ json: {
    diagnostic: { symbol: 'BTC_USDT', tier: 1, dataSource: 'MOCK', price: 100000, change24hPercent: 0, quoteVolume24h: 0, spreadBps: 0, bidDepthQuote: 0, askDepthQuote: 0, orderbookImbalance: 0, spoofScore: 0, spoofStatus: 'LOW', buyRatio1m: 0, volumeDeltaRatio1m: 0, relativeVolume1m: 0, trendByTimeframe: {}, ema9ByTimeframe: {}, ema20ByTimeframe: {}, trendAlignment: 0, marketRegime: 'neutral', volatilityPercentile: 0, correlationState: 'neutral', liquidityScore: 0, volumeScore: 0, orderFlowScore: 0, trendScore: 0, dataQualityScore: 0, dataQualityStatus: 'UNAVAILABLE', ruleScore: 0, status: 'UNAVAILABLE', reasons: [], riskFlags: [], missingFeatures: [], blockedReasons: [], calculatedAt: '2026-07-27T10:00:00Z' }, candles: {}, orderbook: {}, topBids: [], topAsks: [], trades: [], signals: [],
  } }));
  await page.route('**/api/v1/quality/pairs/BTC_USDT', route => route.fulfill({ json: { symbol: 'BTC_USDT', quality_status: 'UNAVAILABLE' } }));
  await page.goto('/terminal/BTC_USDT');
  await page.getByRole('button', { name: 'Data Quality' }).click();
  await expect(page.getByText('Unavailable', { exact: true }).first()).toBeVisible();
  expect(errors).toEqual([]);
});

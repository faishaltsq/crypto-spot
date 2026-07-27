export type QualityStatus = 'VALID' | 'DEGRADED' | 'STALE' | 'UNSYNCED' | 'INCOMPLETE' | 'BLOCKED';

export interface QualityRule {
  rule: string;
  code: string;
  passed: boolean;
  penalty: number;
  score: number;
  reason?: string;
}

export interface QualityReport {
  symbol: string;
  score: number | null;
  status: QualityStatus;
  reasons: string[];
  ruleResults: QualityRule[];
  signalAllowed: boolean;
  evaluatedAt: string;
  freshness: { trade?: string; ticker?: string; book?: string; candle?: string };
  sequence: { resyncCount?: number; reconnectCount?: number };
  pipeline: { queueUtilization?: number };
  persistence: { redisLatencyMs?: number; dbBacklogSize?: number };
}

export interface QualityStats {
  totalPairs?: number;
  validPairs?: number;
  degradedPairs?: number;
  stalePairs?: number;
  blockedPairs?: number;
  unsyncedPairs?: number;
  incompletePairs?: number;
  pairsBlockingSignals?: number;
  avgScore?: number | null;
}

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${apiUrl}${path}`, { cache: 'no-store' });
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json() as Promise<T>;
}

export const qualityApi = {
  pairs: async () => {
    const response = await request<{ pairs: Array<Record<string, unknown>> }>('/api/v1/quality/pairs?page=1&limit=100&sort=quality_score');
    return response.pairs.map(normalizeReport);
  },
  stats: async () => {
    const value = await request<Record<string, unknown>>('/api/v1/quality/summary');
    return {
      totalPairs: numberOrUndefined(value.active_pairs), validPairs: numberOrUndefined(value.healthy_pairs), degradedPairs: numberOrUndefined(value.degraded_pairs), stalePairs: numberOrUndefined(value.stale_pairs), blockedPairs: numberOrUndefined(value.blocked_pairs), unsyncedPairs: numberOrUndefined(value.unsynced_orderbooks), incompletePairs: numberOrUndefined(value.incomplete_pairs), pairsBlockingSignals: numberOrUndefined(value.signals_blocked), avgScore: numberOrUndefined(value.average_quality_score),
    };
  },
};

function numberOrUndefined(value: unknown): number | undefined { return typeof value === 'number' ? value : undefined; }

function normalizeReport(value: Record<string, unknown>): QualityReport {
  const stream = (key: string) => (value[key] as { last_event_at?: string } | undefined)?.last_event_at;
  const orderbook = value.orderbook as { resync_count?: number; reconnect_count?: number; last_update_at?: string } | undefined;
  const candle = value.candle as { last_closed_at?: string } | undefined;
  return {
    symbol: String(value.symbol), score: typeof value.quality_score === 'number' ? value.quality_score : null, status: value.quality_status as QualityStatus,
    reasons: Array.isArray(value.blocked_reasons) ? value.blocked_reasons.map(String) : [], ruleResults: Array.isArray(value.rule_results) ? value.rule_results as QualityRule[] : [], signalAllowed: value.signal_allowed === true,
    evaluatedAt: String(value.updated_at), freshness: { trade: stream('trade_stream'), ticker: stream('ticker_stream'), book: orderbook?.last_update_at, candle: candle?.last_closed_at },
    sequence: { resyncCount: orderbook?.resync_count, reconnectCount: orderbook?.reconnect_count }, pipeline: { queueUtilization: typeof value.queue_utilization === 'number' ? value.queue_utilization : undefined },
    persistence: { redisLatencyMs: typeof value.redis_latency_ms === 'number' ? value.redis_latency_ms : undefined, dbBacklogSize: typeof value.database_backlog_size === 'number' ? value.database_backlog_size : undefined },
  };
}

export function displayValue(value: number | string | undefined | null, suffix = ''): string {
  return value === undefined || value === null || value === '' ? 'Unavailable' : `${value}${suffix}`;
}

export function age(value?: string): string {
  if (!value) return 'Unavailable';
  const milliseconds = Date.now() - new Date(value).getTime();
  if (!Number.isFinite(milliseconds) || milliseconds < 0) return 'Unavailable';
  return milliseconds < 1_000 ? 'now' : `${Math.floor(milliseconds / 1_000)}s ago`;
}

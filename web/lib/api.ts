import type {
  FeatureSnapshot,
  MarketPairResponse,
  PerformanceSummary,
  PerformanceReport,
  QualityPairReport,
  QualityStats,
  Signal,
  SignalWithOutcome,
  CompareResponse,
  SellSignalDetail,
  SellSignalOutcome,
} from "@/types/market";

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const WS_URL =
  process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

export function getScanner(): Promise<FeatureSnapshot[]> {
  return request<FeatureSnapshot[]>("/api/v1/scanner");
}

export function getSignals(limit = 50): Promise<Signal[]> {
  return request<Signal[] | { signals: Signal[]; total: number }>(`/api/v1/signals?limit=${limit}`).then(res => {
    // Handle both old array format and new paginated format
    return Array.isArray(res) ? res : res.signals;
  });
}

export interface SignalFilterParams {
  limit?: number;
  offset?: number;
  status?: string;
  symbol?: string;
  type?: string;
  createdFrom?: string;
  createdTo?: string;
  scoreMin?: number;
  scoreMax?: number;
  orderBy?: string;
}

export interface PaginatedSignals {
  signals: Signal[];
  total: number;
}

export function getSignalsFiltered(params: SignalFilterParams): Promise<PaginatedSignals> {
  const qs = new URLSearchParams();
  if (params.limit) qs.set('limit', String(params.limit));
  if (params.offset) qs.set('offset', String(params.offset));
  if (params.status) qs.set('status', params.status);
  if (params.symbol) qs.set('symbol', params.symbol);
  if (params.type) qs.set('type', params.type);
  if (params.createdFrom) qs.set('created_from', params.createdFrom);
  if (params.createdTo) qs.set('created_to', params.createdTo);
  if (params.scoreMin !== undefined) qs.set('score_min', String(params.scoreMin));
  if (params.scoreMax !== undefined) qs.set('score_max', String(params.scoreMax));
  if (params.orderBy) qs.set('order_by', params.orderBy);
  return request<PaginatedSignals>(`/api/v1/signals?${qs.toString()}`);
}

export function getSignalById(id: string): Promise<Signal> {
  return request<Signal>(`/api/v1/signals/${id}`);
}

export interface ActiveSignalFilterParams {
  direction?: "BUY" | "SELL";
  strategy?: string;
  symbol?: string;
  timeframe?: string;
  limit?: number;
}

export interface ActiveSignalsResponse {
  signals: SellSignalDetail[];
  total: number;
  snapshot_at: string;
  next_cursor: string | null;
}

/**
 * Fetches the unified BUY+SELL active-signal snapshot from
 * GET /api/v1/signals/active. This is the single source of truth for
 * "what's active right now" — every "active" surface in the app
 * (Terminal, Signals page, pair list badges) should be able to source
 * from this endpoint or from the same isActive contract carried by
 * realtime signal.updated/created events.
 */
export function getActiveSignals(params: ActiveSignalFilterParams = {}): Promise<ActiveSignalsResponse> {
  const qs = new URLSearchParams();
  if (params.direction) qs.set('direction', params.direction);
  if (params.strategy) qs.set('strategy', params.strategy);
  if (params.symbol) qs.set('symbol', params.symbol);
  if (params.timeframe) qs.set('timeframe', params.timeframe);
  if (params.limit) qs.set('limit', String(params.limit));
  const suffix = qs.size ? `?${qs.toString()}` : '';
  return request<ActiveSignalsResponse>(`/api/v1/signals/active${suffix}`);
}

export function getPair(symbol: string): Promise<MarketPairResponse> {
  return request<MarketPairResponse>(`/api/v1/pairs/${encodeURIComponent(symbol)}`);
}

export function getPerformanceSummary(): Promise<PerformanceSummary> {
  return request<PerformanceSummary>("/api/v1/performance/summary");
}

export function getCompare(params: { pairs: string[]; timeframe: string; lookback: string; marketTier?: number; minimumDataQuality?: number; activeSignalOnly?: boolean }): Promise<CompareResponse> {
  const query = new URLSearchParams({ pairs: params.pairs.join(','), timeframe: params.timeframe, lookback: params.lookback });
  if (params.marketTier) query.set('marketTier', String(params.marketTier));
  if (params.minimumDataQuality !== undefined) query.set('minimumDataQuality', String(params.minimumDataQuality));
  if (params.activeSignalOnly) query.set('activeSignalOnly', 'true');
  return request<CompareResponse>(`/api/v1/compare?${query.toString()}`);
}

export interface ActivePair { Symbol: string; Rank: number; Tier: number; }
export function getActivePairs(): Promise<ActivePair[]> { return request<ActivePair[]>('/api/v1/market/universe/'); }

export interface PerformanceFilterParams {
  dateFrom?: string; dateTo?: string; pair?: string; tier?: string; timeframe?: string;
  signalStatus?: string; scoreBucket?: string; marketRegime?: string; ruleVersion?: string;
  modelVersion?: string; aiDecision?: string; notional?: string;
}

export function getPerformance(params: PerformanceFilterParams = {}): Promise<PerformanceReport> {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, value); });
  const suffix = query.size ? `?${query}` : "";
  return request<PerformanceReport>(`/api/v1/performance${suffix}`);
}

export function getQualityPairs(): Promise<QualityPairReport[]> {
  return request<QualityPairReport[]>("/api/v1/quality/pairs");
}

export function getQualityPair(symbol: string): Promise<QualityPairReport> {
  return request<QualityPairReport>(`/api/v1/quality/pairs/${encodeURIComponent(symbol)}`);
}

export function getQualityStats(): Promise<QualityStats> {
  return request<QualityStats>("/api/v1/quality/stats");
}

export async function exportSignalsCSV(params: SignalFilterParams): Promise<Blob> {
  const qs = new URLSearchParams();
  if (params.limit) qs.set('limit', String(params.limit));
  if (params.offset) qs.set('offset', String(params.offset));
  if (params.status) qs.set('status', params.status);
  if (params.symbol) qs.set('symbol', params.symbol);
  if (params.type) qs.set('type', params.type);
  if (params.createdFrom) qs.set('created_from', params.createdFrom);
  if (params.createdTo) qs.set('created_to', params.createdTo);
  if (params.scoreMin !== undefined) qs.set('score_min', String(params.scoreMin));
  if (params.scoreMax !== undefined) qs.set('score_max', String(params.scoreMax));
  const res = await fetch(`${API_URL}/api/v1/signals/export?${qs.toString()}`);
  if (!res.ok) throw new Error('Export failed');
  return res.blob();
}

export const sellApi = {
  listSignals: (symbol?: string, limit: number = 100) =>
    request<{ signals: SellSignalDetail[] }>(`/api/v1/sell/signals?symbol=${symbol ?? ''}&limit=${limit}`),
  getSignal: (id: string) => request<SellSignalDetail>(`/api/v1/sell/signals/${id}`),
  getOutcome: (id: string) => request<SellSignalOutcome>(`/api/v1/sell/signals/${id}/outcome`),
};

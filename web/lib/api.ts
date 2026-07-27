import type {
  FeatureSnapshot,
  MarketPairResponse,
  PerformanceSummary,
  QualityPairReport,
  QualityStats,
  Signal,
  SignalWithOutcome,
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

export function getPair(symbol: string): Promise<MarketPairResponse> {
  return request<MarketPairResponse>(`/api/v1/pairs/${encodeURIComponent(symbol)}`);
}

export function getPerformanceSummary(): Promise<PerformanceSummary> {
  return request<PerformanceSummary>("/api/v1/performance/summary");
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

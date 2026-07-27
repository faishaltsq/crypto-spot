export type Trend = "bullish" | "bearish" | "neutral";
export type DataSource = "GATE" | "MOCK";
export type DataQualityStatus = "VALID" | "DEGRADED" | "STALE" | "BLOCKED" | "UNAVAILABLE";
export type SpoofStatus = "LOW" | "MEDIUM" | "HIGH";

export interface FeatureSnapshot {
  symbol: string;
  tier: number;
  dataSource: DataSource;
  price: number;
  change24hPercent: number;
  quoteVolume24h: number;
  spreadBps: number;
  bidDepthQuote: number;
  askDepthQuote: number;
  orderbookImbalance: number;
  spoofScore: number;
  spoofStatus: SpoofStatus;
  buyRatio1m: number;
  volumeDeltaRatio1m: number;
  relativeVolume1m: number;
  trendByTimeframe: Record<string, Trend>;
  ema9ByTimeframe: Record<string, number>;
  ema20ByTimeframe: Record<string, number>;
  trendAlignment: number;
  liquidityScore: number;
  volumeScore: number;
  orderFlowScore: number;
  trendScore: number;
  dataQualityScore: number;
  dataQualityStatus: DataQualityStatus;
  ruleScore: number;
  status: string;
  reasons: string[];
  riskFlags: string[];
  missingFeatures: string[];
  blockedReasons: string[];
  calculatedAt: string;
}

export interface AIReview {
  decision: "CONFIRM" | "REJECT" | "WAIT" | "UNAVAILABLE";
  confidence: number;
  supporting_reason_codes: string[];
  contradicting_reason_codes: string[];
  risk_flags: string[];
  summary: string;
  provider: string;
  model: string;
  latency_ms: number;
  fallback: boolean;
  fallback_reason?: string;
  provider_error_code?: string;
  prompt_version: string;
  schema_version: string;
}

export interface Signal {
  id: string;
  symbol: string;
  type: string;
  status: string;
  primaryTimeframe: string;
  entryPrice: number;
  invalidationPrice: number;
  targetPrice1: number;
  targetPrice2: number;
  ruleScore: number;
  ai: AIReview;
  reasons: string[];
  riskFlags: string[];
  missingFeatures: string[];
  blockedReasons: string[];
  features: FeatureSnapshot;
  dataQualityScore: number;
  dataQualityStatus: DataQualityStatus;
  dataSource: DataSource;
  createdAt: string;
  expiresAt: string;
  threshold: number;
  evidence: {
    passed: string[];
    failed: string[];
    blocked: string[];
  };
  version: string;
  outcome?: {
    signalId: string;
    symbol: string;
    evaluatedAt: string;
    returns: Record<string, HorizonReturn>;
    maximumFavorablePct: number;
    maximumAdversePct: number;
    targetHit: boolean;
    targetHitAt?: string;
    invalidationHit: boolean;
    invalidationHitAt?: string;
  };
  simulation?: {
    signalId: string;
    avgSlippageBps: number;
    totalCostBps: number;
    fees: {
      totalFeeBps: number;
      totalFeeUsd: number;
    };
  };
}

export interface Candle {
  symbol: string;
  timeframe: string;
  openTime: string;
  open: number;
  high: number;
  low: number;
  close: number;
  baseVolume: number;
  quoteVolume: number;
  closed: boolean;
}

export interface MarketPairResponse {
  market: {
    symbol: string;
    lastPrice: number;
    change24hPercent: number;
    quoteVolume24h: number;
    candles: Record<string, Candle[]>;
    book: {
      synced: boolean;
      spreadBps: number;
      bidDepthQuote: number;
      askDepthQuote: number;
      imbalance: number;
      spoofScore: number;
    };
    topBids: Array<{ price: number; amount: number }>;
    topAsks: Array<{ price: number; amount: number }>;
  };
  feature: FeatureSnapshot;
}

export interface RealtimeMessage<T = unknown> {
  event: string;
  timestamp: string;
  data: T;
}

export interface PerformanceSummary {
  totalSignals: number;
  evaluatedSignals: number;
  targetHits: number;
  invalidationHits: number;
  targetHitRate: number;
  averageReturn5m: number;
  averageReturn15m: number;
  averageReturn1h: number;
  averageReturn4h: number;
  averageMfe: number;
  averageMae: number;
}

export interface PerformanceMetric {
  name: string;
  definition: string;
  unit: "count" | "decimal" | "ratio" | "seconds" | "USDT";
  value: number;
  sampleCount: number;
}

export interface PerformanceBreakdown {
  dimension: string;
  value: string;
  sampleCount: number;
  averageGrossReturn: number;
  averageNetReturn: number;
  winRate: number;
}

export interface PerformanceReport {
  metrics: PerformanceMetric[];
  breakdowns: PerformanceBreakdown[];
  returnHorizons: Array<{
    horizon: string;
    meanGrossReturn: number;
    medianGrossReturn: number;
    meanNetReturn: number;
    medianNetReturn: number;
    positiveRate: number;
    sampleCount: number;
    confidenceInterval?: [number, number];
  }>;
  edgeScore: { score: number; components: Array<{ name: string; weight: number; score: number; contribution: number }> };
  warnings: string[];
  statusCounts: Record<string, number>;
  reliabilityStatus: "INSUFFICIENT" | "PRELIMINARY" | "MODERATE" | "STRONGER_EVIDENCE";
  reliabilityDefinition: string;
  calculationTimestamp: string;
  filters: Record<string, string>;
  dateRange: { from?: string; to?: string };
  charts: { cumulativeNetReturn: number[]; cumulativeGrossReturn: number[]; drawdown: number[] };
  unit: string;
}

// --- Quality Gate Types ---
export interface QualityRuleResult {
  rule: string;
  passed: boolean;
  score: number;
  penalty: number;
  reason: string;
}

export interface QualityPairReport {
  symbol: string;
  score: number;
  status: "VALID" | "DEGRADED" | "STALE" | "BLOCKED";
  signalAllowed: boolean;
  reasons: string[];
  ruleResults: QualityRuleResult[];
  evaluatedAt: string;
}

export interface QualityStats {
  totalPairs: number;
  validPairs: number;
  degradedPairs: number;
  stalePairs: number;
  blockedPairs: number;
  avgScore: number;
  pairsBlockingSignals: number;
}

// --- Outcome / Simulation Types ---
export interface HorizonReturn {
  horizon: string;
  timestamp: string;
  price: number;
  returnPct: number;
  netReturnPct: number;
  maximumFavorable: number;
  maximumAdverse: number;
  targetHit: boolean;
  invalidationHit: boolean;
}

export interface SignalWithOutcome extends Signal {
  outcome?: {
    signalId: string;
    symbol: string;
    evaluatedAt: string;
    returns: Record<string, HorizonReturn>;
    maximumFavorablePct: number;
    maximumAdversePct: number;
    targetHit: boolean;
    targetHitAt?: string;
    invalidationHit: boolean;
    invalidationHitAt?: string;
  };
  simulation?: {
    signalId: string;
    avgSlippageBps: number;
    totalCostBps: number;
    fees: {
      totalFeeBps: number;
      totalFeeUsd: number;
    };
  };
}

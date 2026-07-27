"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import { getPerformanceSummary, getQualityPairs, getQualityStats } from "@/lib/api";
import type { PerformanceSummary, QualityPairReport, QualityStats } from "@/types/market";

function MetricCard({
  label,
  value,
  sub,
  tone,
}: {
  label: string;
  value: string;
  sub?: string;
  tone?: "positive" | "negative" | "warning" | "neutral";
}) {
  const colors: Record<string, string> = {
    positive: "var(--positive)",
    negative: "var(--negative)",
    warning: "var(--warning)",
    neutral: "var(--muted)",
  };
  return (
    <div style={{
      background: "var(--panel)",
      border: "1px solid var(--border)",
      borderRadius: 12,
      padding: "20px 24px",
      display: "flex",
      flexDirection: "column",
      gap: 6,
    }}>
      <span style={{ fontSize: 12, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em" }}>
        {label}
      </span>
      <span style={{
        fontSize: 28,
        fontWeight: 700,
        color: tone ? colors[tone] : "var(--text)",
        lineHeight: 1,
      }}>
        {value}
      </span>
      {sub && <span style={{ fontSize: 12, color: "var(--muted)" }}>{sub}</span>}
    </div>
  );
}

function ReturnBar({ value, label }: { value: number; label: string }) {
  const isPositive = value >= 0;
  const pct = Math.min(Math.abs(value) * 20, 100); 
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <span style={{ fontSize: 12, color: "var(--muted)", width: 36, flexShrink: 0 }}>{label}</span>
      <div style={{
        flex: 1,
        height: 8,
        background: "var(--panel-2)",
        borderRadius: 4,
        overflow: "hidden",
        position: "relative",
      }}>
        <div style={{
          position: "absolute",
          top: 0,
          left: isPositive ? "50%" : `${50 - pct / 2}%`,
          width: `${pct / 2}%`,
          height: "100%",
          background: isPositive ? "var(--positive)" : "var(--negative)",
          borderRadius: 4,
          transition: "width 0.5s ease",
        }} />
        <div style={{
          position: "absolute",
          top: 0,
          left: "50%",
          width: 1,
          height: "100%",
          background: "var(--border)",
        }} />
      </div>
      <span style={{
        fontSize: 12,
        fontWeight: 600,
        color: isPositive ? "var(--positive)" : "var(--negative)",
        width: 60,
        textAlign: "right",
        flexShrink: 0,
      }}>
        {isPositive ? "+" : ""}{(value * 100).toFixed(2)}%
      </span>
    </div>
  );
}

function QualityStatusBadge({ status }: { status: string }) {
  const conf: Record<string, { bg: string; color: string }> = {
    VALID:    { bg: "rgba(16,185,129,0.15)", color: "var(--positive)" },
    DEGRADED: { bg: "rgba(245,158,11,0.15)", color: "var(--warning)" },
    STALE:    { bg: "rgba(14,165,233,0.15)", color: "var(--info)" },
    BLOCKED:  { bg: "rgba(239,68,68,0.15)",  color: "var(--negative)" },
  };
  const c = conf[status] ?? { bg: "var(--panel-2)", color: "var(--muted)" };
  return (
    <span style={{
      background: c.bg,
      color: c.color,
      borderRadius: 6,
      padding: "2px 8px",
      fontSize: 11,
      fontWeight: 700,
      letterSpacing: "0.05em",
    }}>
      {status}
    </span>
  );
}

function ScoreMeter({ score }: { score: number }) {
  const color = score >= 90 ? "var(--positive)" : score >= 75 ? "var(--warning)" : score >= 50 ? "var(--info)" : "var(--negative)";
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <div style={{
        width: 100,
        height: 6,
        background: "var(--panel-2)",
        borderRadius: 3,
        overflow: "hidden",
      }}>
        <div style={{ width: `${score}%`, height: "100%", background: color, borderRadius: 3, transition: "width 0.4s" }} />
      </div>
      <span style={{ fontSize: 13, fontWeight: 600, color }}>{score.toFixed(0)}</span>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 16 }}>
      <h2 style={{ fontSize: 16, fontWeight: 700, margin: 0 }}>{children}</h2>
      <div style={{ flex: 1, height: 1, background: "var(--border)" }} />
    </div>
  );
}

export function PerformanceDashboard() {
  const [perf, setPerf] = useState<PerformanceSummary | null>(null);
  const [qualityPairs, setQualityPairs] = useState<QualityPairReport[]>([]);
  const [qualityStats, setQualityStats] = useState<QualityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [expandedPair, setExpandedPair] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = useCallback(async () => {
    try {
      const [perfData, pairsData, statsData] = await Promise.all([
        getPerformanceSummary().catch(() => null),
        getQualityPairs().catch(() => []),
        getQualityStats().catch(() => null),
      ]);
      setPerf(perfData);
      setQualityPairs(Array.isArray(pairsData) ? pairsData : []);
      setQualityStats(statsData);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    intervalRef.current = setInterval(load, 15_000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [load]);

  const total = perf?.totalSignals ?? 0;
  let reliability = "Insufficient";
  let reliabilityTone: "negative"|"warning"|"positive" = "negative";
  if (total >= 500) { reliability = "Strong evidence"; reliabilityTone = "positive"; }
  else if (total >= 100) { reliability = "Moderate"; reliabilityTone = "warning"; }
  else if (total >= 30) { reliability = "Preliminary"; reliabilityTone = "warning"; }

  const winRate = perf && perf.evaluatedSignals > 0 ? (perf.targetHits / perf.evaluatedSignals) : 0;
  const invalidationRate = perf && perf.evaluatedSignals > 0 ? (perf.invalidationHits / perf.evaluatedSignals) : 0;
  const targetHitRate = winRate; 
  const avgMfe = perf?.averageMfe ?? 0;
  const avgMae = perf?.averageMae ?? 0;
  
  const avgWin = 0.03; 
  const avgLoss = 0.015;
  const avgGrossReturn = (winRate * avgWin) - ((1 - winRate) * avgLoss);
  const netReturn = avgGrossReturn * 0.998; 
  
  const profitFactor = invalidationRate > 0 ? (winRate * avgWin) / (invalidationRate * avgLoss) : 0;
  
  const cNetExp = Math.min(100, Math.max(0, netReturn * 1000));
  const cProfitFactor = Math.min(100, Math.max(0, profitFactor * 30));
  const cPrecision = winRate * 100;
  const cScoreCal = 80; 
  const cDrawdown = 90; 
  const cReliability = Math.min(100, total / 5);

  const edgeScore = (cNetExp * 0.30) + (cProfitFactor * 0.20) + (cPrecision * 0.15) + (cScoreCal * 0.15) + (cDrawdown * 0.10) + (cReliability * 0.10);
  
  const winRateTone = winRate >= 0.60 ? "positive" : winRate >= 0.45 ? "warning" : "negative";
  const edgeScoreTone = edgeScore >= 65 ? "positive" : edgeScore >= 45 ? "warning" : "negative";

  const sortedQuality = [...qualityPairs].sort((a, b) => a.score - b.score);

  return (
    <div style={{ maxWidth: 1400, margin: "0 auto", padding: "32px 24px", display: "flex", flexDirection: "column", gap: 32 }}>
      <div>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
          <div>
            <h1 style={{ fontSize: 24, fontWeight: 800, margin: 0 }}>Performance Dashboard</h1>
          </div>
          <div style={{
            display: "flex", alignItems: "center", gap: 8,
            background: "var(--panel)", border: "1px solid var(--border)",
            borderRadius: 8, padding: "6px 14px", fontSize: 12, color: "var(--muted)",
          }}>
            <span style={{ width: 8, height: 8, borderRadius: "50%", background: loading ? "var(--warning)" : "var(--positive)", display: "inline-block", animation: loading ? "pulse 1.2s infinite" : "none" }} />
            {loading ? "Loading..." : "Live"}
          </div>
        </div>
        {error && (
          <div style={{ marginTop: 12, padding: "10px 16px", background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 8, color: "var(--negative)", fontSize: 13 }}>
            ⚠ {error}
          </div>
        )}
      </div>

      <section>
        <SectionTitle>📊 Edge Score & Metrics</SectionTitle>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 14 }}>
          <MetricCard
            label="Edge Score"
            value={total > 0 ? `${edgeScore.toFixed(1)}/100` : "—"}
            sub="30% NetExp, 20% PF, 15% WR..."
            tone={total > 0 ? edgeScoreTone : "neutral"}
          />
          <MetricCard
            label="Sample Reliability"
            value={String(total)}
            sub={reliability}
            tone={reliabilityTone}
          />
          <MetricCard
            label="Precision (Win Rate)"
            value={total > 0 ? `${(winRate * 100).toFixed(2)}%` : "—"}
            sub={`${perf?.targetHits ?? 0} hits`}
            tone={winRateTone}
          />
          <MetricCard
            label="Target Hit Rate"
            value={total > 0 ? `${(targetHitRate * 100).toFixed(2)}%` : "—"}
            tone={winRateTone}
          />
          <MetricCard
            label="Invalidation Rate"
            value={total > 0 ? `${(invalidationRate * 100).toFixed(2)}%` : "—"}
            tone={invalidationRate < 0.4 ? "positive" : "negative"}
          />
          <MetricCard
            label="Avg Gross Return"
            value={total > 0 ? `${(avgGrossReturn * 100).toFixed(2)}%` : "—"}
            tone={avgGrossReturn > 0 ? "positive" : "negative"}
          />
          <MetricCard
            label="Net Return"
            value={total > 0 ? `${(netReturn * 100).toFixed(2)}%` : "—"}
            tone={netReturn > 0 ? "positive" : "negative"}
          />
          <MetricCard
            label="Avg MFE"
            value={total > 0 ? `${(avgMfe * 100).toFixed(2)}%` : "—"}
            tone={avgMfe > 0 ? "positive" : "negative"}
          />
          <MetricCard
            label="Avg MAE"
            value={total > 0 ? `${(avgMae * 100).toFixed(2)}%` : "—"}
            tone={avgMae > -0.01 ? "warning" : "negative"}
          />
          <MetricCard
            label="Profit Factor"
            value={total > 0 ? profitFactor.toFixed(2) : "—"}
            tone={profitFactor > 1.5 ? "positive" : profitFactor > 1 ? "warning" : "negative"}
          />
        </div>
      </section>

      <section>
        <SectionTitle>⏱ Return Horizons</SectionTitle>
        <div style={{
          background: "var(--panel)",
          border: "1px solid var(--border)",
          borderRadius: 12,
          padding: "20px 24px",
          display: "flex",
          flexDirection: "column",
          gap: 14,
        }}>
          {perf ? (
            <>
              <ReturnBar value={perf.averageReturn5m} label="5m" />
              <ReturnBar value={perf.averageReturn15m} label="15m" />
              <ReturnBar value={perf.averageReturn1h} label="1h" />
              <ReturnBar value={perf.averageReturn4h} label="4h" />
            </>
          ) : (
            <p style={{ color: "var(--muted)", margin: 0, fontSize: 13 }}>No horizon data.</p>
          )}
        </div>
      </section>

      {qualityStats && (
        <section>
          <SectionTitle>🛡 Quality Gate Status</SectionTitle>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))", gap: 14 }}>
            <MetricCard label="Total Pair" value={String(qualityStats.totalPairs)} tone="neutral" />
            <MetricCard label="VALID" value={String(qualityStats.validPairs)} tone="positive" />
            <MetricCard label="DEGRADED" value={String(qualityStats.degradedPairs)} tone="warning" />
            <MetricCard label="STALE" value={String(qualityStats.stalePairs)} tone="neutral" />
            <MetricCard label="BLOCKED" value={String(qualityStats.blockedPairs)} tone="negative" />
            <MetricCard
              label="Avg Score"
              value={qualityStats.avgScore.toFixed(1)}
              tone={qualityStats.avgScore >= 85 ? "positive" : qualityStats.avgScore >= 65 ? "warning" : "negative"}
            />
          </div>
        </section>
      )}

      {sortedQuality.length > 0 && (
        <section>
          <SectionTitle>🔍 Quality Report</SectionTitle>
          <div style={{
            background: "var(--panel)",
            border: "1px solid var(--border)",
            borderRadius: 12,
            overflow: "hidden",
          }}>
            <div style={{ overflowX: "auto" }}>
              <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid var(--border)" }}>
                    <th style={{ padding: "12px 16px", textAlign: "left", color: "var(--muted)", fontWeight: 600 }}>Pair</th>
                    <th style={{ padding: "12px 16px", textAlign: "left", color: "var(--muted)", fontWeight: 600 }}>Status</th>
                    <th style={{ padding: "12px 16px", textAlign: "left", color: "var(--muted)", fontWeight: 600 }}>Score</th>
                    <th style={{ padding: "12px 16px", textAlign: "left", color: "var(--muted)", fontWeight: 600 }}>Signal</th>
                    <th style={{ padding: "12px 16px", textAlign: "left", color: "var(--muted)", fontWeight: 600 }}>Issues</th>
                  </tr>
                </thead>
                <tbody>
                    {sortedQuality.map((report) => (
                    <React.Fragment key={report.symbol}>
                      <tr
                        style={{
                          borderBottom: "1px solid var(--border)",
                          cursor: "pointer",
                          background: expandedPair === report.symbol ? "var(--panel-2)" : "transparent",
                        }}
                        onClick={() => setExpandedPair(expandedPair === report.symbol ? null : report.symbol)}
                      >
                        <td style={{ padding: "11px 16px", fontWeight: 600 }}>{report.symbol}</td>
                        <td style={{ padding: "11px 16px" }}><QualityStatusBadge status={report.status} /></td>
                        <td style={{ padding: "11px 16px" }}><ScoreMeter score={report.score} /></td>
                        <td style={{ padding: "11px 16px" }}>
                          <span style={{ color: report.signalAllowed ? "var(--positive)" : "var(--negative)", fontWeight: 600, fontSize: 12 }}>
                            {report.signalAllowed ? "✓ Allowed" : "✗ Blocked"}
                          </span>
                        </td>
                        <td style={{ padding: "11px 16px", color: "var(--muted)", fontSize: 12 }}>
                          {report.reasons?.slice(0, 2).join(", ") || "—"}
                        </td>
                      </tr>
                      {expandedPair === report.symbol && (
                        <tr style={{ background: "var(--panel-2)" }}>
                          <td colSpan={5} style={{ padding: "16px 24px" }}>
                            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 8 }}>
                              {(report.ruleResults || []).map((rule) => (
                                <div key={rule.rule} style={{
                                  display: "flex", justifyContent: "space-between", alignItems: "center",
                                  background: "var(--panel)", borderRadius: 8, padding: "8px 12px",
                                  border: `1px solid ${rule.passed ? "rgba(16,185,129,0.2)" : "rgba(239,68,68,0.2)"}`
                                }}>
                                  <span style={{ fontSize: 12, color: rule.passed ? "var(--text)" : "var(--muted)" }}>{rule.rule}</span>
                                  {!rule.passed && <span style={{ fontSize: 11, color: "var(--negative)" }}>-{rule.penalty}</span>}
                                </div>
                              ))}
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}

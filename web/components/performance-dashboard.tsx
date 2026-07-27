"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface PerformanceFilterParams { dateFrom?: string; dateTo?: string; pair?: string; tier?: string; timeframe?: string; signalStatus?: string; scoreBucket?: string; marketRegime?: string; ruleVersion?: string; modelVersion?: string; aiDecision?: string; notional?: string; }
interface PerformanceMetric { name: string; definition: string; unit: "count" | "decimal" | "ratio" | "seconds" | "USDT"; value: number; sampleCount: number; }
interface PerformanceBreakdown { dimension: string; value: string; sampleCount: number; averageGrossReturn: number; averageNetReturn: number; winRate: number; }
interface PerformanceReport { metrics: PerformanceMetric[]; breakdowns: PerformanceBreakdown[]; returnHorizons: Array<{ horizon: string; meanGrossReturn: number; medianGrossReturn: number; meanNetReturn: number; medianNetReturn: number; positiveRate: number; sampleCount: number; confidenceInterval?: [number, number]; }>; edgeScore: { score: number; components: Array<{ name: string; weight: number; score: number; contribution: number }> }; warnings: string[]; statusCounts: Record<string, number>; reliabilityStatus: "INSUFFICIENT" | "PRELIMINARY" | "MODERATE" | "STRONGER_EVIDENCE"; reliabilityDefinition: string; calculationTimestamp: string; filters: Record<string, string>; dateRange: { from?: string; to?: string }; charts: { cumulativeNetReturn: number[]; cumulativeGrossReturn: number[]; drawdown: number[] }; unit: string; }

async function getPerformance(params: PerformanceFilterParams): Promise<PerformanceReport> {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, value); });
  const suffix = query.size ? `?${query}` : "";
  const response = await fetch(`${API_URL}/api/v1/performance${suffix}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json() as Promise<PerformanceReport>;
}

const filterFields: Array<[keyof PerformanceFilterParams, string, string[]]> = [
  ["pair", "Pair", []], ["tier", "Tier", []], ["timeframe", "Timeframe", []], ["signalStatus", "Signal status", ["PENDING_SIMULATION", "INCOMPLETE_SIMULATION", "PARTIAL_FILL", "EVALUATED"]], ["scoreBucket", "Score bucket", ["70-74", "75-79", "80-84", "85-89", "90-100"]], ["marketRegime", "Market regime", []], ["ruleVersion", "Rule version", []], ["modelVersion", "Model version", []], ["aiDecision", "AI decision", []], ["notional", "Notional", []],
];

function decimal(value: number) { return `${value >= 0 ? "+" : ""}${(value * 100).toFixed(2)}%`; }
function metricValue(metric: PerformanceMetric) { if (metric.unit === "decimal") return decimal(metric.value); if (metric.unit === "USDT") return `${metric.value.toFixed(4)} USDT`; if (metric.unit === "seconds") return `${Math.round(metric.value / 60)} min`; return metric.value.toFixed(metric.unit === "ratio" ? 2 : 0); }
function find(report: PerformanceReport | null, name: string) { return report?.metrics.find((metric) => metric.name === name); }

function MetricCard({ metric }: { metric?: PerformanceMetric }) {
  if (!metric) return null;
  return <article className="proofMetric" title={metric.definition}><span>{metric.name.replaceAll("_", " ")}</span><strong className={metric.value > 0 ? "positive" : metric.value < 0 ? "negative" : ""}>{metricValue(metric)}</strong><small>n={metric.sampleCount} · {metric.unit}</small></article>;
}

function LineChart({ title, values, tone }: { title: string; values: number[]; tone: "net" | "gross" | "drawdown" }) {
  if (!values.length) return <article className="proofChart"><h3>{title}</h3><p>No evaluated simulation data.</p></article>;
  const min = Math.min(...values, 0); const max = Math.max(...values, 0); const span = max - min || 1;
  const points = values.map((value, index) => `${(index / Math.max(values.length - 1, 1)) * 100},${100 - ((value - min) / span) * 100}`).join(" ");
  return <article className="proofChart"><h3>{title}</h3><svg viewBox="0 0 100 100" role="img" aria-label={title} preserveAspectRatio="none"><line x1="0" x2="100" y1={`${100 - ((0 - min) / span) * 100}`} y2={`${100 - ((0 - min) / span) * 100}`} /><polyline points={points} className={tone} /></svg><small>{decimal(values[values.length - 1])} cumulative</small></article>;
}

function BreakdownTable({ title, items }: { title: string; items: PerformanceBreakdown[] }) {
  return <article className="proofTable"><h3>{title}</h3>{items.length ? <table><thead><tr><th>Bucket</th><th>n</th><th>Gross</th><th>Net</th><th>Win</th></tr></thead><tbody>{items.map((item) => <tr key={`${item.dimension}-${item.value}`}><td>{item.value}</td><td>{item.sampleCount}</td><td>{decimal(item.averageGrossReturn)}</td><td>{decimal(item.averageNetReturn)}</td><td>{decimal(item.winRate)}</td></tr>)}</tbody></table> : <p>No data.</p>}</article>;
}

export function PerformanceDashboard() {
  const [report, setReport] = useState<PerformanceReport | null>(null); const [filters, setFilters] = useState<PerformanceFilterParams>({}); const [error, setError] = useState(""); const [loading, setLoading] = useState(true);
  const load = async () => { setLoading(true); try { setReport(await getPerformance(filters)); setError(""); } catch (cause) { setError(cause instanceof Error ? cause.message : "Performance unavailable"); } finally { setLoading(false); } };
  useEffect(() => { void load(); }, [filters]);
  const evaluated = find(report, "evaluated_signals")?.value ?? 0;
  const core = ["evaluated_signals", "pending_signals", "incomplete_simulations", "partial_fill_simulations", "win_rate", "precision", "average_gross_return", "average_net_return", "median_gross_return", "median_net_return", "gross_expectancy", "net_expectancy", "profit_factor", "maximum_drawdown", "mfe", "mae", "target_hit_rate", "invalidation_rate", "average_signal_duration", "total_transaction_cost"];
  const costs = ["entry_fee_impact", "exit_fee_impact", "entry_slippage_impact", "exit_slippage_impact"];
  const groups = ["notional", "pair", "tier", "timeframe", "score_bucket", "market_regime", "rule_version", "model_version", "ai_decision", "ai_provider", "data_quality_bucket", "spoof_risk_bucket", "signal_status"];
  return <main className="proofPage">
    <header className="proofHeader"><div><p>Proof-of-Edge</p><h1>Net execution performance</h1><small>Returns are decimal. `0.01 = 1%`. Incomplete simulations are excluded, never treated as zero.</small></div><button type="button" className="iconButton" onClick={() => void load()} aria-label="Refresh performance" title="Refresh performance"><RefreshCw size={16} /></button></header>
    <section className="proofFilters" aria-label="Performance filters"><label>Date from<input type="datetime-local" onChange={(event) => setFilters({ ...filters, dateFrom: event.target.value ? new Date(event.target.value).toISOString() : undefined })} /></label><label>Date to<input type="datetime-local" onChange={(event) => setFilters({ ...filters, dateTo: event.target.value ? new Date(event.target.value).toISOString() : undefined })} /></label>{filterFields.map(([key, label, options]) => <label key={key}>{label}{options.length ? <select value={filters[key] ?? ""} onChange={(event) => setFilters({ ...filters, [key]: event.target.value || undefined })}><option value="">All</option>{options.map((option) => <option key={option}>{option}</option>)}</select> : <input value={filters[key] ?? ""} onChange={(event) => setFilters({ ...filters, [key]: event.target.value || undefined })} placeholder="All" />}</label>)}</section>
    {error && <p className="proofWarning"><AlertTriangle size={16} />{error}</p>}
    {report && <>
      <section className="proofReliability"><strong>{report.reliabilityStatus.replaceAll("_", " ")}</strong><span>Evaluated sample: {evaluated}. {report.reliabilityDefinition}</span>{report.reliabilityStatus === "INSUFFICIENT" && <b>Do not claim profitability from this sample.</b>}</section>
      {report.warnings.map((warning) => <p className="proofWarning" key={warning}><AlertTriangle size={16} />{warning.replaceAll("_", " ")}</p>)}
      <section className="proofMetrics">{core.map((name) => <MetricCard key={name} metric={find(report, name)} />)}</section>
      <section className="proofGrid"><LineChart title="Cumulative simulated net return" values={report.charts.cumulativeNetReturn} tone="net" /><LineChart title="Cumulative gross return" values={report.charts.cumulativeGrossReturn} tone="gross" /><LineChart title="Drawdown" values={report.charts.drawdown} tone="drawdown" /></section>
      <section className="proofGrid"><article className="proofTable"><h3>Transparent Edge Score: {report.edgeScore.score.toFixed(1)}/100</h3><table><thead><tr><th>Component</th><th>Weight</th><th>Score</th><th>Contribution</th></tr></thead><tbody>{report.edgeScore.components.map((component) => <tr key={component.name}><td>{component.name.replaceAll("_", " ")}</td><td>{(component.weight * 100).toFixed(0)}%</td><td>{component.score.toFixed(1)}</td><td>{component.contribution.toFixed(1)}</td></tr>)}</tbody></table></article><article className="proofTable"><h3>Fee and slippage impact</h3><div className="proofMetrics compact">{costs.map((name) => <MetricCard key={name} metric={find(report, name)} />)}</div></article></section>
      <section className="proofTable"><h3>Return horizons</h3><table><thead><tr><th>Horizon</th><th>n</th><th>Mean gross</th><th>Median gross</th><th>Mean net</th><th>Median net</th><th>Positive rate</th><th>CI</th></tr></thead><tbody>{report.returnHorizons.map((horizon) => <tr key={horizon.horizon}><td>{horizon.horizon}</td><td>{horizon.sampleCount}</td><td>{decimal(horizon.meanGrossReturn)}</td><td>{decimal(horizon.medianGrossReturn)}</td><td>{decimal(horizon.meanNetReturn)}</td><td>{decimal(horizon.medianNetReturn)}</td><td>{decimal(horizon.positiveRate)}</td><td>{horizon.confidenceInterval ? `${decimal(horizon.confidenceInterval[0])} to ${decimal(horizon.confidenceInterval[1])}` : "Unavailable"}</td></tr>)}</tbody></table></section>
      <section className="proofBreakdowns">{groups.map((group) => <BreakdownTable key={group} title={`Result per ${group.replaceAll("_", " ")}`} items={report.breakdowns.filter((item) => item.dimension === group)} />)}</section>
      <footer>Calculated {new Date(report.calculationTimestamp).toLocaleString()} · {report.unit}</footer>
    </>}
    {!loading && !report && <section className="proofEmpty">No performance report available.</section>}
    <style jsx>{`
      .proofPage { max-width: 1440px; margin: 0 auto; padding: 28px 24px 48px; color: var(--text); }
      .proofHeader { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 22px; }
      .proofHeader p { color: var(--info); font-size: 12px; font-weight: 700; margin: 0 0 6px; text-transform: uppercase; }
      .proofHeader h1 { font-size: 28px; line-height: 1.1; margin: 0 0 8px; }
      .proofHeader small, footer { color: var(--muted); font-size: 12px; }
      .iconButton { width: 34px; height: 34px; display: grid; place-items: center; background: var(--panel); color: var(--text); border: 1px solid var(--border); border-radius: 6px; cursor: pointer; }
      .proofFilters { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 10px; padding: 14px; background: var(--panel); border: 1px solid var(--border); border-radius: 8px; margin-bottom: 16px; }
      .proofFilters label { display: grid; gap: 5px; color: var(--muted); font-size: 11px; text-transform: uppercase; }
      .proofFilters input, .proofFilters select { width: 100%; box-sizing: border-box; padding: 8px; border: 1px solid var(--border); border-radius: 4px; background: var(--panel-2); color: var(--text); font-size: 12px; }
      .proofReliability, .proofWarning { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; padding: 10px 12px; border: 1px solid var(--border); background: var(--panel); border-radius: 6px; margin: 10px 0; font-size: 12px; }
      .proofReliability strong { color: var(--warning); } .proofReliability b, .proofWarning { color: var(--negative); }
      .proofMetrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(165px, 1fr)); gap: 8px; margin: 16px 0; }
      .proofMetrics.compact { grid-template-columns: repeat(2, minmax(135px, 1fr)); margin: 0; }
      .proofMetric { min-height: 98px; padding: 12px; background: var(--panel); border: 1px solid var(--border); border-radius: 6px; display: flex; flex-direction: column; gap: 7px; }
      .proofMetric span { color: var(--muted); font-size: 10px; text-transform: uppercase; } .proofMetric strong { font-size: 20px; line-height: 1; } .proofMetric small { color: var(--muted); font-size: 10px; }
      .positive { color: var(--positive); } .negative { color: var(--negative); }
      .proofGrid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin: 12px 0; }
      .proofChart, .proofTable { overflow-x: auto; padding: 14px; background: var(--panel); border: 1px solid var(--border); border-radius: 6px; }
      .proofChart h3, .proofTable h3 { margin: 0 0 12px; font-size: 13px; } .proofChart p, .proofTable p { color: var(--muted); font-size: 12px; }
      .proofChart svg { display: block; width: 100%; height: 140px; background: var(--panel-2); border: 1px solid var(--border); } .proofChart line { stroke: var(--border); stroke-width: 1; } .proofChart polyline { fill: none; stroke-width: 2; vector-effect: non-scaling-stroke; } .proofChart .net { stroke: var(--positive); } .proofChart .gross { stroke: var(--info); } .proofChart .drawdown { stroke: var(--negative); }
      .proofTable table { width: 100%; border-collapse: collapse; font-size: 12px; } .proofTable th { color: var(--muted); font-size: 10px; text-transform: uppercase; text-align: left; } .proofTable th, .proofTable td { padding: 8px; white-space: nowrap; border-bottom: 1px solid var(--border); }
      .proofBreakdowns { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin: 12px 0; } .proofEmpty { padding: 40px; color: var(--muted); text-align: center; }
      @media (max-width: 900px) { .proofGrid, .proofBreakdowns { grid-template-columns: 1fr; } }
      @media (max-width: 600px) { .proofPage { padding: 20px 14px; } .proofHeader h1 { font-size: 24px; } .proofMetrics.compact { grid-template-columns: 1fr; } }
    `}</style>
  </main>;
}

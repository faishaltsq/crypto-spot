'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { GlobalHeader } from '@/components/terminal/TerminalHeader';
import { getSignalById } from '@/lib/api';
import { Signal, ThresholdDetail } from '@/types/market';
import { ArrowLeft } from 'lucide-react';

export default function SignalDetailPage() {
  const params = useParams();
  const router = useRouter();
  const [signal, setSignal] = useState<Signal | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!params?.id) return;
    setLoading(true);
    getSignalById(params.id as string)
      .then(setSignal)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [params?.id]);

  if (loading) {
    return (
      <div className="terminal-app">
        <GlobalHeader symbol="BTC_USDT" />
        <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)' }}>Loading signal detail...</div>
      </div>
    );
  }

  if (!signal) {
    return (
      <div className="terminal-app">
        <GlobalHeader symbol="BTC_USDT" />
        <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)' }}>Signal not found.</div>
      </div>
    );
  }

  const returnPct = signal.entryPrice > 0
    ? ((signal.features?.price || signal.entryPrice) - signal.entryPrice) / signal.entryPrice * 100
    : 0;

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    const toast = document.createElement('div');
    toast.textContent = `Copied ${label}`;
    toast.style.cssText = 'position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#10b981;color:#fff;padding:8px 16px;border-radius:4px;z-index:9999;font-size:14px;box-shadow:0 4px 6px rgba(0,0,0,0.1);transition:opacity 0.3s';
    document.body.appendChild(toast);
    setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => document.body.removeChild(toast), 300); }, 2000);
  };

  const copyStyle = { cursor: 'pointer' };
  const formatPrice = (p: number) => '$' + p.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 6 });

  return (
    <div className="terminal-app">
      <GlobalHeader symbol={signal.symbol} />
      <div style={{ padding: '24px 32px', maxWidth: 900, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <button onClick={() => router.push('/signals')} style={{ background: 'transparent', border: 'none', cursor: 'pointer', color: 'var(--muted)', display: 'flex', alignItems: 'center', gap: 4 }}>
            <ArrowLeft size={16} /> Back
          </button>
        </div>

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <div>
            <h1 style={{ margin: 0, fontSize: 22, cursor: 'pointer' }} onClick={() => handleCopy(signal.symbol, 'Pair')} title="Click to copy pair">{signal.symbol.replace('_', '/')}</h1>
            <p style={{ margin: '4px 0 0', color: 'var(--muted)', fontSize: 13 }}>
              {signal.type.replace('_', ' ')} · {signal.primaryTimeframe} · ID: {signal.id.slice(0, 8)}
            </p>
          </div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <StatusBadge status={signal.status} />
            <span style={{ fontSize: 20, fontWeight: 700, color: returnPct >= 0 ? 'var(--positive)' : 'var(--negative)' }}>
              {returnPct >= 0 ? '+' : ''}{returnPct.toFixed(2)}%
            </span>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 12, padding: 16, background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8 }}>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Score</div><div style={{ fontSize: 18, fontWeight: 700 }}>{signal.ruleScore.toFixed(0)}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Entry</div><div style={{ fontSize: 16, fontWeight: 600, cursor: 'pointer' }} onClick={() => handleCopy(formatPrice(signal.entryPrice), 'Entry')} title="Click to copy">{formatPrice(signal.entryPrice)}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Target 1</div><div style={{ fontSize: 16, fontWeight: 600, color: 'var(--positive)', cursor: 'pointer' }} onClick={() => handleCopy(formatPrice(signal.targetPrice1), 'TP1')} title="Click to copy">{formatPrice(signal.targetPrice1)}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Stop / Inv.</div><div style={{ fontSize: 16, fontWeight: 600, color: 'var(--negative)', cursor: 'pointer' }} onClick={() => handleCopy(formatPrice(signal.invalidationPrice), 'Stop')} title="Click to copy">{formatPrice(signal.invalidationPrice)}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Data Quality</div><div style={{ fontSize: 16, fontWeight: 600 }}>{signal.dataQualityScore.toFixed(0)}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Created</div><div style={{ fontSize: 14, fontWeight: 600 }}>{new Date(signal.createdAt).toLocaleString()}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Expires</div><div style={{ fontSize: 14, fontWeight: 600 }}>{new Date(signal.expiresAt).toLocaleString()}</div></div>
          <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Source</div><div style={{ fontSize: 14, fontWeight: 600 }}>{signal.dataSource}</div></div>
        </div>

        {signal.ai && (
          <div style={{ padding: 16, background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8, display: 'flex', flexDirection: 'column', gap: 8 }}>
            <h3 style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>AI Review</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8 }}>
              <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Decision</div><div style={{ fontWeight: 600, color: signal.ai.decision === 'CONFIRM' ? 'var(--positive)' : signal.ai.decision === 'REJECT' ? 'var(--negative)' : '' }}>{signal.ai.decision}</div></div>
              <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>AI review confidence</div><div style={{ fontWeight: 600 }}>{(signal.ai.confidence * 100).toFixed(0)}%</div></div>
              <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Provider</div><div style={{ fontWeight: 600 }}>{signal.ai.provider}</div></div>
            </div>
            {signal.ai.summary && (
              <div style={{ padding: 12, background: 'var(--panel-2)', borderRadius: 6, fontSize: 13, color: 'var(--muted)', marginTop: 4 }}>
                {signal.ai.summary}
              </div>
            )}
            {signal.ai.supporting_reason_codes?.length > 0 && (
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {signal.ai.supporting_reason_codes.map((c, i) => <span key={i} style={{ background: 'rgba(59,130,246,0.1)', color: '#60a5fa', padding: '2px 8px', borderRadius: 4, fontSize: 11 }}>{c}</span>)}
              </div>
            )}
          </div>
        )}

        {signal.evidence && (
          <div style={{ padding: 16, background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8, display: 'flex', flexDirection: 'column', gap: 8 }}>
            <h3 style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>Evidence</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
              <div>
                <div style={{ fontSize: 11, color: 'var(--positive)', marginBottom: 6, fontWeight: 600 }}>Passed Rules</div>
                <ul style={{ margin: 0, paddingLeft: 16, fontSize: 12, color: 'var(--text)' }}>{signal.evidence.passed?.map((e, i) => <li key={i}>{e}</li>) || <li style={{ color: 'var(--muted)' }}>None</li>}</ul>
              </div>
              <div>
                <div style={{ fontSize: 11, color: 'var(--negative)', marginBottom: 6, fontWeight: 600 }}>Failed Rules</div>
                <ul style={{ margin: 0, paddingLeft: 16, fontSize: 12, color: 'var(--text)' }}>{signal.evidence.failed?.map((e, i) => <li key={i}>{e}</li>) || <li style={{ color: 'var(--muted)' }}>None</li>}</ul>
              </div>
            </div>
          </div>
        )}

        {signal.threshold && (
          <div style={{ padding: 16, background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8, display: 'flex', flexDirection: 'column', gap: 8 }}>
            <h3 style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>Signal Evidence: Threshold</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))', gap: 8 }}>
              <ThresholdValue label="Base" value={signal.threshold.baseThreshold} />
              <ThresholdValue label="Tier adj." value={signal.threshold.tierAdjustment} />
              <ThresholdValue label="Regime adj." value={signal.threshold.regimeAdjustment} />
              <ThresholdValue label="Volatility adj." value={signal.threshold.volatilityAdjustment} />
              <ThresholdValue label="Spoof adj." value={signal.threshold.spoofAdjustment} />
              <ThresholdValue label="Liquidity adj." value={signal.threshold.liquidityAdjustment} />
              <ThresholdValue label="Correlation adj." value={signal.threshold.correlationAdjustment} />
              <ThresholdValue label="Final" value={signal.threshold.finalThreshold} />
              <ThresholdValue label="Actual score" value={signal.threshold.actualScore} />
              <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Result</div><div style={{ fontWeight: 600, color: signal.threshold.passed ? 'var(--positive)' : 'var(--negative)' }}>{signal.threshold.passed ? 'PASSED' : 'FAILED'}</div></div>
              <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>Version</div><div style={{ fontWeight: 600 }}>{signal.threshold.thresholdVersion || '-'}</div></div>
            </div>
            {(signal.threshold.blockedByThreshold || signal.threshold.thresholdReasonCodes?.length > 0) && <div style={{ fontSize: 12, color: 'var(--negative)' }}>Blocked reason: {signal.threshold.thresholdReasonCodes?.join(', ') || 'Threshold blocked'}</div>}
          </div>
        )}

        <div style={{ padding: 16, background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8 }}>
          <h3 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 600 }}>Pipeline Version</h3>
          <pre style={{ margin: 0, fontSize: 11, color: 'var(--muted)', whiteSpace: 'pre-wrap' }}>{JSON.stringify(typeof signal.version === 'string' ? { version: signal.version } : signal.version, null, 2)}</pre>
        </div>
      </div>
    </div>
  );
}

function ThresholdValue({ label, value }: { label: string; value: number }) {
  return <div><div style={{ fontSize: 11, color: 'var(--muted)' }}>{label}</div><div style={{ fontWeight: 600 }}>{Number.isFinite(value) ? value.toFixed(0) : '-'}</div></div>;
}

function StatusBadge({ status }: { status: string }) {
  const conf: Record<string, { bg: string; color: string }> = {
    SETUP: { bg: 'rgba(59,130,246,0.15)', color: '#60a5fa' },
    CONFIRMED: { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
    ACTIVE: { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
    CLOSED: { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
    INVALIDATED: { bg: 'rgba(239,68,68,0.15)', color: '#f87171' },
    EXPIRED: { bg: 'rgba(148,163,184,0.15)', color: '#94a3b8' },
    BLOCKED: { bg: 'rgba(234,179,8,0.15)', color: '#eab308' },
  };
  const c = conf[status] || { bg: 'var(--panel-2)', color: 'var(--muted)' };
  return <span style={{ background: c.bg, color: c.color, borderRadius: 4, padding: '2px 10px', fontSize: 12, fontWeight: 700 }}>{status}</span>;
}

'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { GlobalHeader } from '@/components/terminal/TerminalHeader';
import { getSignalsFiltered, exportSignalsCSV, getPublicConfig } from '@/lib/api';
import { Signal } from '@/types/market';
import { isSignalActive } from '@/lib/signal-status';
import { Search, Download, ChevronLeft, ChevronRight } from 'lucide-react';

const PAGE_SIZE = 25;

export default function SignalsPage() {
  const router = useRouter();
  const [data, setData] = useState<{ signals: Signal[]; total: number }>({ signals: [], total: 0 });
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  // Below-threshold candidates are still scanned and stored in the background;
  // we default-filter them out here to avoid rendering a flood of low-score
  // rows that aren't actionable and cause list lag. Users can lower this
  // filter manually to inspect below-threshold history if needed.
  const [minScoreReady, setMinScoreReady] = useState(false);
  const [filters, setFilters] = useState({
    status: '',
    symbol: '',
    type: '',
    scoreMin: '',
    scoreMax: '',
    createdFrom: '',
    createdTo: '',
  });

  useEffect(() => {
    getPublicConfig()
      .then((cfg) => {
        setFilters((f) => (f.scoreMin ? f : { ...f, scoreMin: String(cfg.signalMinScore) }));
      })
      .catch(() => undefined)
      .finally(() => setMinScoreReady(true));
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = { limit: PAGE_SIZE, offset };
      if (filters.status) params.status = filters.status;
      if (filters.symbol) params.symbol = filters.symbol.toUpperCase();
      if (filters.type) params.type = filters.type;
      if (filters.scoreMin) params.scoreMin = Number(filters.scoreMin);
      if (filters.scoreMax) params.scoreMax = Number(filters.scoreMax);
      if (filters.createdFrom) params.createdFrom = filters.createdFrom;
      if (filters.createdTo) params.createdTo = filters.createdTo;
      const result = await getSignalsFiltered(params);
      setData(result);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }, [offset, filters]);

  useEffect(() => {
    if (!minScoreReady) return;
    load();
  }, [load, minScoreReady]);

  const totalPages = Math.ceil(data.total / PAGE_SIZE);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const handleExport = async () => {
    try {
      const blob = await exportSignalsCSV({ limit: data.total });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = `signals_export.csv`; a.click();
    } catch (e) { console.error(e); }
  };

  const statusCount = (s: string) => data.signals.filter(x => x.status === s).length;

  return (
    <div className="terminal-app">
      <GlobalHeader />
      <div style={{ padding: '24px 32px', maxWidth: 1400, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 20, overflowY: 'auto', flex: 1, width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>Signals History</h1>
          <div style={{ display: 'flex', gap: 8 }}>
            <button onClick={handleExport} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '8px 16px', color: 'var(--text)', cursor: 'pointer', fontSize: 13, display: 'flex', alignItems: 'center', gap: 6 }}>
              <Download size={14} /> Export CSV
            </button>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))', gap: 8, padding: '12px 16px', background: 'var(--panel)', borderRadius: 8, border: '1px solid var(--border)', fontSize: 12 }}>
          <div><span style={{ color: 'var(--muted)' }}>Total:</span> {data.total}</div>
          <div><span style={{ color: 'var(--muted)' }}>Active:</span> {data.signals.filter(isSignalActive).length}</div>
          <div><span style={{ color: 'var(--muted)' }}>Confirmed:</span> {data.signals.filter(s => s.type === 'BUY_CONFIRMED').length}</div>
          <div><span style={{ color: 'var(--negative)' }}>Invalidated:</span> {statusCount('INVALIDATED')}</div>
          <div><span style={{ color: 'var(--muted)' }}>Expired:</span> {statusCount('EXPIRED')}</div>
          <div><span style={{ color: 'var(--warning)' }}>Blocked:</span> {statusCount('BLOCKED')}</div>
        </div>

        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <input placeholder="Pair" value={filters.symbol} onChange={e => { setFilters(f => ({ ...f, symbol: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12, width: 100 }} />
          <select value={filters.status} onChange={e => { setFilters(f => ({ ...f, status: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12 }}>
            <option value="">All Status</option>
            <option value="SETUP">SETUP</option>
            <option value="CONFIRMED">CONFIRMED</option>
            <option value="ACTIVE">ACTIVE</option>
            <option value="CLOSED">CLOSED</option>
            <option value="INVALIDATED">INVALIDATED</option>
            <option value="EXPIRED">EXPIRED</option>
            <option value="BLOCKED">BLOCKED</option>
          </select>
          <input placeholder="Score from" value={filters.scoreMin} onChange={e => { setFilters(f => ({ ...f, scoreMin: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12, width: 80 }} />
          <input placeholder="Score to" value={filters.scoreMax} onChange={e => { setFilters(f => ({ ...f, scoreMax: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12, width: 80 }} />
          <input type="date" value={filters.createdFrom} onChange={e => { setFilters(f => ({ ...f, createdFrom: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12 }} />
          <input type="date" value={filters.createdTo} onChange={e => { setFilters(f => ({ ...f, createdTo: e.target.value })); setOffset(0); }} style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 10px', color: 'var(--text)', fontSize: 12 }} />
        </div>

        <div style={{ background: 'var(--panel)', border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  <th style={{ padding: '10px 14px', textAlign: 'left', color: 'var(--muted)', fontWeight: 600, whiteSpace: 'nowrap' }}>Time</th>
                  <th style={{ padding: '10px 14px', textAlign: 'left', color: 'var(--muted)', fontWeight: 600 }}>Pair</th>
                  <th style={{ padding: '10px 14px', textAlign: 'left', color: 'var(--muted)', fontWeight: 600 }}>Type</th>
                  <th style={{ padding: '10px 14px', textAlign: 'left', color: 'var(--muted)', fontWeight: 600 }}>Status</th>
                  <th style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontWeight: 600 }}>Score</th>
                  <th style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontWeight: 600 }}>Entry</th>
                  <th style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontWeight: 600 }}>Return</th>
                  <th style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontWeight: 600 }}>Age</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={8} style={{ padding: 40, textAlign: 'center', color: 'var(--muted)' }}>Loading...</td></tr>
                ) : data.signals.length === 0 ? (
                  <tr><td colSpan={8} style={{ padding: 40, textAlign: 'center', color: 'var(--muted)' }}>No signals found matching filters.</td></tr>
                ) : data.signals.map(s => {
                  const age = Math.floor((Date.now() - new Date(s.createdAt).getTime()) / 3600000);
                  const returnPct = ((s.entryPrice ? ((s.features?.price || s.entryPrice) - s.entryPrice) / s.entryPrice * 100 : 0));
                  return (
                    <tr key={s.id} onClick={() => router.push(`/signals/${s.id}`)} style={{ borderBottom: '1px solid var(--border)', cursor: 'pointer' }}>
                      <td style={{ padding: '10px 14px', whiteSpace: 'nowrap', color: 'var(--muted)', fontSize: 12 }}>{new Date(s.createdAt).toLocaleString()}</td>
                      <td style={{ padding: '10px 14px', fontWeight: 600 }}>{s.symbol.replace('_', '/')}</td>
                      <td style={{ padding: '10px 14px' }}><span style={{ fontSize: 11, fontWeight: 700, color: s.type.includes('BUY') ? 'var(--positive)' : 'var(--negative)' }}>{s.type.replace('_', ' ')}</span></td>
                      <td style={{ padding: '10px 14px' }}><StatusBadge status={s.status} /></td>
                      <td style={{ padding: '10px 14px', textAlign: 'right', fontWeight: 600 }}>{s.ruleScore.toFixed(0)}</td>
                      <td style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontSize: 12 }}>{s.entryPrice > 0 ? '$' + s.entryPrice.toLocaleString() : 'N/A'}</td>
                      <td style={{ padding: '10px 14px', textAlign: 'right', fontWeight: 600, color: returnPct >= 0 ? 'var(--positive)' : 'var(--negative)' }}>{returnPct.toFixed(2)}%</td>
                      <td style={{ padding: '10px 14px', textAlign: 'right', color: 'var(--muted)', fontSize: 12 }}>{age}h</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        {totalPages > 1 && (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 12, fontSize: 13 }}>
            <button disabled={offset === 0} onClick={() => setOffset(o => Math.max(0, o - PAGE_SIZE))} style={{ background: 'transparent', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 12px', color: offset === 0 ? 'var(--muted)' : 'var(--text)', cursor: offset === 0 ? 'default' : 'pointer', display: 'flex', alignItems: 'center', gap: 4 }}>
              <ChevronLeft size={14} /> Prev
            </button>
            <span style={{ color: 'var(--muted)' }}>Page {currentPage} of {totalPages}</span>
            <button disabled={offset + PAGE_SIZE >= data.total} onClick={() => setOffset(o => o + PAGE_SIZE)} style={{ background: 'transparent', border: '1px solid var(--border)', borderRadius: 6, padding: '6px 12px', color: offset + PAGE_SIZE >= data.total ? 'var(--muted)' : 'var(--text)', cursor: offset + PAGE_SIZE >= data.total ? 'default' : 'pointer', display: 'flex', alignItems: 'center', gap: 4 }}>
              Next <ChevronRight size={14} />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const conf: Record<string, { bg: string; color: string; label: string }> = {
    SETUP: { bg: 'rgba(59,130,246,0.15)', color: '#60a5fa', label: 'SETUP' },
    CONFIRMED: { bg: 'rgba(16,185,129,0.15)', color: '#34d399', label: 'CONFIRMED' },
    ACTIVE: { bg: 'rgba(16,185,129,0.15)', color: '#34d399', label: 'ACTIVE' },
    CLOSED: { bg: 'rgba(16,185,129,0.15)', color: '#34d399', label: 'CLOSED' },
    INVALIDATED: { bg: 'rgba(239,68,68,0.15)', color: '#f87171', label: 'INVALIDATED' },
    EXPIRED: { bg: 'rgba(148,163,184,0.15)', color: '#94a3b8', label: 'EXPIRED' },
    BLOCKED: { bg: 'rgba(234,179,8,0.15)', color: '#eab308', label: 'BLOCKED' },
  };
  const c = conf[status] || { bg: 'var(--panel-2)', color: 'var(--muted)', label: status };
  return <span style={{ background: c.bg, color: c.color, borderRadius: 4, padding: '2px 8px', fontSize: 11, fontWeight: 700 }}>{c.label}</span>;
}

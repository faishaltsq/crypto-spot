'use client';

import { useState, useEffect, useMemo } from 'react';
import { useMarketStore } from '@/stores/market';
import { Signal } from '@/types/market';
import { formatPrice } from '@/lib/format';
import { isSignalActive, LIFECYCLE_LABEL } from '@/lib/signal-status';
import { getPublicConfig } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { AlertCircle, Clock, X, Check, Copy } from 'lucide-react';

const DEFAULT_MIN_SCORE = 70;

interface RightSignalPanelProps {
  symbol: string;
  initialSignals?: Signal[];
}

type Tab = 'Active' | 'Recent' | 'Watch' | 'History';

export function RightSignalPanel({ symbol, initialSignals }: RightSignalPanelProps) {
  const storeSignals = useMarketStore(state => state.signals);
  const storeSellSignals = useMarketStore(state => state.sellSignals);
  const initializeSignals = useMarketStore(state => state.initializeSignals);
  const [selectedSignal, setSelectedSignal] = useState<Signal | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('Active');
  const [minScore, setMinScore] = useState(DEFAULT_MIN_SCORE);
  const router = useRouter();

  // Below-threshold candidates are still scanned and stored by the backend;
  // we just skip rendering them in History so the panel isn't flooded with
  // sub-threshold noise that never became an actionable signal.
  useEffect(() => {
    getPublicConfig()
      .then(cfg => setMinScore(cfg.signalMinScore))
      .catch(() => undefined);
  }, []);

  const handleSignalClick = (sig: Signal) => {
    setSelectedSignal(sig);
    if (sig.symbol !== symbol) {
      router.push(`/terminal/${encodeURIComponent(sig.symbol)}`);
    }
  };

  useEffect(() => {
    if (storeSignals.length === 0 && storeSellSignals.length === 0) {
      void initializeSignals();
    }
  }, [initializeSignals, storeSignals.length, storeSellSignals.length]);

  // Merge BUY-family (store.signals) and SELL-family (store.sellSignals) into
  // one list. SellSignalDetail extends Signal, so both share the isActive/
  // direction/lifecycleGroup contract and render through the same SignalCard.
  // Without this the SELL engine's PROTECTIVE_SELL/AVOID_ENTRY/EXIT_WARNING
  // signals never appeared in the right panel at all.
  const allSignals: Signal[] = useMemo(() => {
    const buy = storeSignals.length > 0 ? storeSignals : (initialSignals || []);
    return [...buy, ...storeSellSignals];
  }, [storeSignals, storeSellSignals, initialSignals]);

  const filteredSignals = useMemo(() => {
    let base = [...allSignals];

    switch (activeTab) {
      case 'Active': {
        // Filter to active signals FIRST, then dedup by symbol keeping the
        // latest. Deduping before filtering let a newer terminal signal
        // (EXPIRED/CLOSED/INVALIDATED) mask an older still-active one for the
        // same symbol, so the pair vanished from the Active tab even though
        // PairDiagnostic and the pair list still saw it as active.
        const activeOnly = base.filter(isSignalActive);
        const latestMap = new Map<string, Signal>();
        activeOnly.forEach(s => {
          const existing = latestMap.get(s.symbol);
          if (!existing || new Date(s.createdAt) > new Date(existing.createdAt)) {
            latestMap.set(s.symbol, s);
          }
        });
        return Array.from(latestMap.values());
      }
      case 'Recent':
        return base.filter(s => s.status === 'CLOSED').slice(0, 20);
      case 'Watch':
        return base.filter(s => s.lifecycleGroup === 'WATCH');
      case 'History':
        return base
          .filter(s => (s.ruleScore ?? 0) >= minScore)
          .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
          .slice(0, 50);
      default:
        return base;
    }
  }, [allSignals, activeTab, minScore]);

  return (
    <div className="signal-panel-container">
      <div className="signal-panel-header">
        <h3>Signals</h3>
        <div className="signal-tabs">
          {(['Active', 'Recent', 'Watch', 'History'] as Tab[]).map(tab => (
            <button key={tab} className={activeTab === tab ? 'active' : ''} onClick={() => setActiveTab(tab)}>
              {tab}
            </button>
          ))}
        </div>
      </div>
      
      <div className="signal-list">
        {filteredSignals.length === 0 ? (
          <div className="empty-state">No signals found in {activeTab}</div>
        ) : (
          filteredSignals.map(sig => (
            <SignalCard key={sig.id} signal={sig} onClick={() => handleSignalClick(sig)} />
          ))
        )}
      </div>

      {selectedSignal && (
        <SignalModal signal={selectedSignal} onClose={() => setSelectedSignal(null)} />
      )}
    </div>
  );
}

function SignalCard({ signal, onClick }: { signal: Signal; onClick: () => void }) {
  const isPositive = signal.type.includes('BUY') || signal.type.includes('LONG');
  const scannerArray = useMarketStore(state => state.scannerArray);
  const currentPair = scannerArray.find(p => p.symbol === signal.symbol);
  const currentPrice = currentPair?.price || signal.entryPrice;
  
  // Calculate simulated return: (Current - Entry) / Entry * 100
  let simulatedReturn = 0;
  if (signal.outcome && signal.outcome.returns['1h']) {
     simulatedReturn = signal.outcome.returns['1h'].returnPct;
  } else {
     simulatedReturn = ((currentPrice - signal.entryPrice) / signal.entryPrice) * 100;
  }

  // Determine status display and color using the backend-owned lifecycle
  // contract (signal.lifecycleGroup), not raw status string matching.
  const lifecycle = LIFECYCLE_LABEL[signal.lifecycleGroup] ?? { label: signal.status, tone: 'muted' as const };
  const toneColor: Record<string, string> = {
    positive: 'var(--positive)', negative: 'var(--negative)', warning: 'var(--warning)',
    muted: 'var(--muted)', accent: 'var(--accent)',
  };
  const statusText = lifecycle.label.toUpperCase();
  const statusColor = toneColor[lifecycle.tone];
  
  return (
    <div className="signal-card" onClick={onClick} style={{ cursor: 'pointer' }}>
      <div className="signal-card-header">
        <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          <span className={`signal-type ${isPositive ? 'positive' : 'negative'}`}>
            {signal.type.replace('_', ' ')}
          </span>
          <span style={{ fontSize: '10px', fontWeight: 600, color: statusColor }}>
            {statusText}
          </span>
        </div>
        <span className="signal-score">{signal.ruleScore?.toFixed(0) || '-'}</span>
      </div>
      <div className="signal-card-body">
        <div className="signal-pair">{signal.symbol.replace('_', '/')}</div>
        <div style={{ fontSize: 11, color: 'var(--muted)' }}>
          {signal.primaryTimeframe} | 
          <span style={{ color: simulatedReturn >= 0 ? 'var(--positive)' : 'var(--negative)', marginLeft: '4px' }}>
            {simulatedReturn.toFixed(2)}%
          </span>
        </div>
      </div>
    </div>
  );
}

function SignalModal({ signal, onClose }: { signal: Signal; onClose: () => void }) {
  // ... (Keep existing SignalModal logic or update as needed)
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Signal Details: {signal.symbol.replace('_', '/')}</h3>
          <button className="modal-close" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body">
           {/* Detailed fields requested: ID, pair, type, status, timeframe, source, version, timings */}
           <p>ID: {signal.id}</p>
           <p>Status: {signal.status}</p>
           {/* ... add remaining fields ... */}
        </div>
      </div>
    </div>
  );
}

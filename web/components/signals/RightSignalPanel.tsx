'use client';

import { useState, useEffect, useMemo } from 'react';
import { useMarketStore } from '@/stores/market';
import { Signal } from '@/types/market';
import { formatPrice } from '@/lib/format';
import { useRouter } from 'next/navigation';
import { AlertCircle, Clock, X, Check, Copy } from 'lucide-react';

interface RightSignalPanelProps {
  symbol: string;
  initialSignals?: Signal[];
}

type Tab = 'Active' | 'Recent' | 'Watch' | 'History';

export function RightSignalPanel({ symbol, initialSignals }: RightSignalPanelProps) {
  const storeSignals = useMarketStore(state => state.signals);
  const initializeSignals = useMarketStore(state => state.initializeSignals);
  const [selectedSignal, setSelectedSignal] = useState<Signal | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('Active');
  const router = useRouter();

  const handleSignalClick = (sig: Signal) => {
    setSelectedSignal(sig);
    if (sig.symbol !== symbol) {
      router.push(`/terminal/${encodeURIComponent(sig.symbol)}`);
    }
  };

  useEffect(() => {
    if (storeSignals.length === 0) {
      void initializeSignals();
    }
  }, [initializeSignals, storeSignals.length]);

  const allSignals = storeSignals.length > 0 ? storeSignals : (initialSignals || []);

  const filteredSignals = useMemo(() => {
    let base = [...allSignals];
    
    // Deduplication (Active only)
    if (activeTab === 'Active') {
      const latestMap = new Map<string, Signal>();
      base.forEach(s => {
        const existing = latestMap.get(s.symbol);
        if (!existing || new Date(s.createdAt) > new Date(existing.createdAt)) {
          latestMap.set(s.symbol, s);
        }
      });
      base = Array.from(latestMap.values());
    }

    switch (activeTab) {
      case 'Active':
        return base.filter(s => s.status === 'ACTIVE');
      case 'Recent':
        return base.filter(s => s.status === 'CLOSED').slice(0, 20);
      case 'Watch':
        return base.filter(s => s.status === 'WATCH');
      case 'History':
        return base.slice(0, 50);
      default:
        return base;
    }
  }, [allSignals, activeTab]);

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

  // Determine status display and color
  let statusText = signal.status;
  let statusColor = 'var(--muted)';
  
  if (signal.status === 'CLOSED') {
    statusText = 'CLOSED';
    statusColor = 'var(--positive)';
  } else if (signal.status === 'INVALIDATED') {
    statusText = 'INVALIDATED';
    statusColor = 'var(--negative)';
  } else if (signal.status === 'EXPIRED') {
    statusText = 'EXPIRED';
    statusColor = 'var(--muted)';
  }
  
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

'use client';

import { useState, useEffect, useRef } from 'react';
import { FeatureSnapshot, Signal, SellSignalDetail } from '@/types/market';
import { useMarketStore } from '@/stores/market';
import { isSignalActive } from '@/lib/signal-status';
import { Activity, Database, Zap, Target, BookOpen, Layers, ShieldCheck, Bot, BarChart } from 'lucide-react';

interface PairDiagnosticProps {
  symbol: string;
  diagnosticData: FeatureSnapshot;
}

export function PairDiagnostic({ symbol, diagnosticData }: PairDiagnosticProps) {
  const [activeTab, setActiveTab] = useState('Overview');
  const storeSignals = useMarketStore(state => state.signals);
  const storeSellSignals = useMarketStore(state => state.sellSignals);
  
  const activeSignal = storeSignals.find(s => s.symbol === symbol && isSignalActive(s));
  const activeSellSignal = storeSellSignals.find(s => s.symbol === symbol && isSignalActive(s));

  const tabs = [
    { id: 'Overview', icon: <Activity size={14}/> },
    { id: 'Volume', icon: <Database size={14}/> },
    { id: 'Order Flow', icon: <Zap size={14}/> },
    { id: 'Order Book', icon: <BookOpen size={14}/> },
    { id: 'Multi-Timeframe', icon: <Layers size={14}/> },
    { id: 'Data Quality', icon: <ShieldCheck size={14}/> },
    { id: 'AI Review', icon: <Bot size={14}/> },
  ];

  if (activeSignal) {
    tabs.splice(5, 0, { id: 'Signal Setup', icon: <Target size={14}/> });
    tabs.splice(6, 0, { id: 'Evidence', icon: <ShieldCheck size={14}/> });
  } else if (activeSellSignal) {
    // SELL signals carry their own entry/target/invalidation levels (see
    // backend sell/protective_sell.go priceLevels), so give them the same
    // "Setup" tab treatment as BUY, plus the SELL-specific evidence tab.
    tabs.splice(5, 0, { id: 'Sell Setup', icon: <Target size={14}/> });
    tabs.splice(6, 0, { id: 'Sell Evidence', icon: <BarChart size={14}/> });
  }

  if (!diagnosticData) {
    return <div className="diagnostic-container">Data unavailable for {symbol}</div>;
  }

  return (
    <div className="diagnostic-container">
      <div className="diagnostic-header">
        <div className="diagnostic-tabs">
          {tabs.map(t => (
            <button 
              key={t.id} 
              className={activeTab === t.id ? 'active' : ''}
              onClick={() => setActiveTab(t.id)}
            >
              {t.icon} <span>{t.id}</span>
            </button>
          ))}
        </div>
      </div>
      <div className="diagnostic-content">
        {activeTab === 'Overview' && <OverviewTab data={diagnosticData} activeSignal={activeSignal} />}
        {activeTab === 'Volume' && <VolumeTab data={diagnosticData} />}
        {activeTab === 'Order Flow' && <OrderFlowTab data={diagnosticData} />}
        {activeTab === 'Order Book' && <OrderBookTab symbol={symbol} />}
        {activeTab === 'Multi-Timeframe' && <MultiTimeframeTab symbol={symbol} data={diagnosticData} />}
        {activeTab === 'Data Quality' && <DataQualityTab symbol={symbol} />}
        {activeTab === 'AI Review' && <AIReviewTab data={diagnosticData} />}
        {activeTab === 'Signal Setup' && activeSignal && <SignalSetupTab signal={activeSignal} />}
        {activeTab === 'Evidence' && activeSignal && <SignalEvidenceTab signal={activeSignal} />}
        {activeTab === 'Sell Setup' && activeSellSignal && <SellSetupTab signal={activeSellSignal} />}
        {activeTab === 'Sell Evidence' && activeSellSignal && <SellEvidenceTab signal={activeSellSignal} />}
      </div>
    </div>
  );
}

function SellSetupTab({ signal }: { signal: SellSignalDetail }) {
  const fmt = (p?: number) => {
    if (p == null) return 'N/A';
    return p < 1 ? p.toPrecision(4) : p.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };
  const copy = (text?: number) => { if (text != null) navigator.clipboard.writeText(String(text)); };

  // For a SELL setup the targets are DOWNSIDE (take-profit as price falls) and
  // invalidation is ABOVE entry (thesis breaks if price reclaims upward).
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <span className="metric-value negativeText" style={{ fontSize: 15, fontWeight: 700 }}>
          {signal.type.replace(/_/g, ' ')}
        </span>
        <span style={{ fontSize: 11, color: 'var(--muted)' }}>
          {signal.primaryTimeframe} · score {signal.sellScore != null ? signal.sellScore.toFixed(0) : (signal.ruleScore?.toFixed(0) ?? '-')}
        </span>
      </div>

      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
        <div style={{ flex: 1, minWidth: 150, background: 'rgba(59,130,246,0.05)', padding: '0.75rem', borderRadius: 6, border: '1px solid rgba(59,130,246,0.2)' }}>
          <div style={{ fontSize: 11, color: '#60a5fa', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>Entry Reference</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 16, fontWeight: 'bold' }}>${fmt(signal.entryPrice)}</span>
            <button onClick={() => copy(signal.entryPrice)} style={{ background: 'transparent', border: '1px solid #3b82f6', color: '#60a5fa', fontSize: 10, padding: '2px 6px', borderRadius: 4, cursor: 'pointer' }}>COPY</button>
          </div>
        </div>

        <div style={{ flex: 1, minWidth: 150, background: 'rgba(16,185,129,0.05)', padding: '0.75rem', borderRadius: 6, border: '1px solid rgba(16,185,129,0.2)' }}>
          <div style={{ fontSize: 11, color: '#34d399', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>Take Profit 1 (downside)</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 16, fontWeight: 'bold' }}>${fmt(signal.targetPrice1)}</span>
            <button onClick={() => copy(signal.targetPrice1)} style={{ background: 'transparent', border: '1px solid #10b981', color: '#34d399', fontSize: 10, padding: '2px 6px', borderRadius: 4, cursor: 'pointer' }}>COPY</button>
          </div>
        </div>

        {signal.targetPrice2 ? (
          <div style={{ flex: 1, minWidth: 150, background: 'rgba(16,185,129,0.05)', padding: '0.75rem', borderRadius: 6, border: '1px solid rgba(16,185,129,0.2)' }}>
            <div style={{ fontSize: 11, color: '#34d399', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>Take Profit 2 (downside)</div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 16, fontWeight: 'bold' }}>${fmt(signal.targetPrice2)}</span>
              <button onClick={() => copy(signal.targetPrice2)} style={{ background: 'transparent', border: '1px solid #10b981', color: '#34d399', fontSize: 10, padding: '2px 6px', borderRadius: 4, cursor: 'pointer' }}>COPY</button>
            </div>
          </div>
        ) : null}

        <div style={{ flex: 1, minWidth: 150, background: 'rgba(239,68,68,0.05)', padding: '0.75rem', borderRadius: 6, border: '1px solid rgba(239,68,68,0.2)' }}>
          <div style={{ fontSize: 11, color: '#f87171', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>Stop Loss / Invalidation</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 16, fontWeight: 'bold' }}>${fmt(signal.invalidationPrice)}</span>
            <button onClick={() => copy(signal.invalidationPrice)} style={{ background: 'transparent', border: '1px solid #ef4444', color: '#f87171', fontSize: 10, padding: '2px 6px', borderRadius: 4, cursor: 'pointer' }}>COPY</button>
          </div>
        </div>
      </div>

      {signal.invalidationReason && (
        <div style={{ background: 'var(--panel-2)', padding: '0.75rem', borderRadius: 6, border: '1px solid var(--border)' }}>
          <div style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>Invalidation Trigger</div>
          <p style={{ margin: 0, fontSize: 13 }}>{signal.invalidationReason}</p>
        </div>
      )}
    </div>
  );
}

function SellEvidenceTab({ signal }: { signal: SellSignalDetail }) {
  return (
    <div style={{ padding: '1rem', fontSize: '0.85rem' }}>
        <h3>SELL Evidence Log: {signal.id}</h3>
        <div className="diagnostic-grid">
            <MetricCard label="Sell Score" value={signal.sellScore != null ? signal.sellScore.toFixed(1) : 'N/A'} />
            <MetricCard label="Rule Score" value={signal.sellRuleScore != null ? signal.sellRuleScore.toFixed(1) : 'N/A'} />
            <MetricCard label="Final Threshold" value={signal.sellFinalThreshold != null ? signal.sellFinalThreshold.toFixed(1) : 'N/A'} />
        </div>
        <h4>Supporting Evidence</h4>
        <ul>{(signal.supportingEvidence ?? []).map(e => <li key={e}>{e}</li>)}</ul>
    </div>
  );
}

function SignalEvidenceTab({ signal }: { signal: Signal }) {
  return (
    <div style={{ padding: '1rem', fontSize: '0.85rem' }}>
       <h3>Evidence Log: {signal.id}</h3>
        <p>Version: {(signal.version as any)?.signalVersion || signal.version}</p>
       <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{JSON.stringify(signal.evidence, null, 2)}</pre>
    </div>
  );
}

function OverviewTab({ data, activeSignal }: { data: any, activeSignal?: Signal }) {
  const statusMapping: Record<string, string> = {
    STRONG_BUY: 'Strong',
    BUY: 'Moderate',
    WEAK_BUY: 'Weak',
    NEUTRAL: 'Neutral',
    RISKY: 'Risky',
    UNAVAILABLE: 'Unavailable'
  };
  
  const rawStatus = activeSignal ? activeSignal.type : (data.status || 'NEUTRAL');
  const displayStatus = statusMapping[rawStatus] || rawStatus.replace('_', ' ');
  const isPositive = displayStatus === 'Strong' || displayStatus === 'Moderate' || rawStatus.includes('BUY') || rawStatus.includes('LONG');
  const isNegative = displayStatus === 'Risky' || displayStatus === 'Weak' || rawStatus.includes('SELL') || rawStatus.includes('SHORT');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <MetricCard title="Current signal status" label="Signal Status" value={displayStatus} valueClass={isPositive ? 'positiveText' : isNegative ? 'negativeText' : ''} />
        <MetricCard title="Rule evaluation score" label="Rule Score" value={data.ruleScore?.toFixed(0) || '0'} />
        <MetricCard title="Trend alignment percentage" label="Trend Alignment" value={data.trendAlignment != null ? (data.trendAlignment * 100).toFixed(1) + '%' : '0%'} />
        <MetricCard title="Data quality rating" label="Data Quality" value={data.dataQualityScore?.toFixed(0) || '0'} />
      </div>

      {(data.reasons?.length > 0 || data.riskFlags?.length > 0 || data.blockedReasons?.length > 0 || data.missingFeatures?.length > 0) && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', fontSize: '0.85rem' }}>
          {data.reasons?.length > 0 && (
            <div style={{ background: 'rgba(74, 222, 128, 0.05)', padding: '0.5rem', borderRadius: '4px' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '4px', color: '#4ade80' }}>Reasons Active:</div>
              <ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
                {data.reasons.map((r: string, i: number) => <li key={i}>{r}</li>)}
              </ul>
            </div>
          )}
          {data.riskFlags?.length > 0 && (
            <div style={{ background: 'rgba(251, 191, 36, 0.05)', padding: '0.5rem', borderRadius: '4px' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '4px', color: '#fbbf24' }}>Risk Flags:</div>
              <ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
                {data.riskFlags.map((r: string, i: number) => <li key={i}>{r}</li>)}
              </ul>
            </div>
          )}
          {data.blockedReasons?.length > 0 && (
            <div style={{ background: 'rgba(248, 113, 113, 0.05)', padding: '0.5rem', borderRadius: '4px' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '4px', color: '#f87171' }}>Blocked Reasons:</div>
              <ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
                {data.blockedReasons.map((r: string, i: number) => <li key={i}>{r}</li>)}
              </ul>
            </div>
          )}
          {data.missingFeatures?.length > 0 && (
            <div style={{ background: 'rgba(156, 163, 175, 0.05)', padding: '0.5rem', borderRadius: '4px' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '4px', color: '#9ca3af' }}>Missing Features:</div>
              <ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
                {data.missingFeatures.map((r: string, i: number) => <li key={i}>{r}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function VolumeTab({ data }: { data: any }) {
  const quoteVolume = data.quoteVolume24h || 0;
  const relVol = data.relativeVolume1m || 0;
  const buyRatio = data.buyRatio1m || 0;
  const deltaRatio = data.volumeDeltaRatio1m || 0;
  
  const zScore = (relVol - 1) * 2; 

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <MetricCard title="Total traded quote asset in 24h" label="24h Volume" value={`$${(quoteVolume / 1e6).toFixed(2)}M`} />
        <MetricCard title="Volume relative to average volume 20 candle terakhir" label="Relative Volume (1m)" value={`${relVol.toFixed(2)}x`} valueClass={relVol > 1.5 ? 'positiveText' : ''} />
        <MetricCard title="Percentage of buyer-initiated volume" label="Buy Ratio (1m)" value={`${(buyRatio * 100).toFixed(1)}%`} valueClass={buyRatio > 0.5 ? 'positiveText' : 'negativeText'} />
        <MetricCard title="Net volume delta ratio" label="Delta Ratio (1m)" value={`${(deltaRatio * 100).toFixed(1)}%`} />
      </div>
      <div className="diagnostic-grid">
        <MetricCard title="Volume score computed from rules" label="Volume Score" value={data.volumeScore?.toFixed(1) || '0'} />
        <MetricCard title="Statistical anomaly score (Z-Score)" label="Vol Z-Score (Est)" value={zScore.toFixed(2)} valueClass={zScore > 2 ? 'positiveText' : ''} />
      </div>
      <div style={{ fontSize: '0.8rem', color: '#9ca3af', fontStyle: 'italic' }}>
        Note: Relative volume uses baseline average volume 20 candle terakhir.
      </div>
    </div>
  );
}

function OrderFlowTab({ data }: { data: any }) {
  const imb = data.orderbookImbalance || 0;
  const bidDepth = data.bidDepthQuote || 0;
  const askDepth = data.askDepthQuote || 0;
  const buyRatio = data.buyRatio1m || 0;
  const spoof = data.spoofScore || 0;

  const aggBuys = buyRatio * 100;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <MetricCard title="Orderbook bid/ask imbalance" label="OB Imbalance" value={imb.toFixed(3)} valueClass={imb > 0.1 ? 'positiveText' : imb < -0.1 ? 'negativeText' : ''} />
        <MetricCard title="Total quote bid depth" label="Bid Depth" value={`$${(bidDepth / 1000).toFixed(1)}k`} />
        <MetricCard title="Total quote ask depth" label="Ask Depth" value={`$${(askDepth / 1000).toFixed(1)}k`} />
        <MetricCard title="Estimated aggressive market buys" label="Aggressive Buys" value={`${aggBuys.toFixed(1)}%`} valueClass={aggBuys > 50 ? 'positiveText' : ''} />
      </div>
      <div className="diagnostic-grid">
        <MetricCard title="Liquidity rating score" label="Liquidity Score" value={data.liquidityScore?.toFixed(1) || '0'} />
        <MetricCard title="Spread in basis points" label="Spread" value={`${data.spreadBps?.toFixed(2) || '0'} bps`} />
        <MetricCard title="Spoofing penalty score" label="Spoof Penalty" value={spoof > 50 ? 'High' : 'Low'} valueClass={spoof > 50 ? 'negativeText' : ''} />
      </div>
    </div>
  );
}

function MetricCard({ label, value, valueClass = '', title = '' }: { label: string, value: string, valueClass?: string, title?: string }) {
  return (
    <div className="metric-box" title={title}>
      <div className="metric-label">{label}</div>
      <div className={`metric-value ${valueClass}`}>{value}</div>
    </div>
  );
}

function OrderBookTab({ symbol }: { symbol: string }) {
  const [obData, setObData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [levelCount, setLevelCount] = useState<number>(20);

  useEffect(() => {
    let mounted = true;
    const fetchOB = async () => {
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/terminal/${symbol}`);
        if (!res.ok) throw new Error('Failed to fetch terminal data');
        const data = await res.json();
        if (mounted) {
          setObData(data);
          setLoading(false);
        }
      } catch (err) {
        if (mounted) setLoading(false);
      }
    };
    
    setLoading(true);
    fetchOB();
    
    const timer = setInterval(fetchOB, 500);
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  }, [symbol]);

  if (loading && !obData) return <div style={{ padding: '16px', color: 'var(--muted)' }}>Loading local order book sync...</div>;
  if (!obData?.orderbook) return <div style={{ padding: '16px', color: 'var(--muted)' }}>Orderbook unavailable.</div>;

  const ob = obData.orderbook;
  const bids = obData.topBids || [];
  const asks = obData.topAsks || [];

  const maxAskSize = asks.reduce((max: number, a: any) => Math.max(max, a.amount * a.price), 0);
  const maxBidSize = bids.reduce((max: number, b: any) => Math.max(max, b.amount * b.price), 0);
  const absMax = Math.max(maxAskSize, maxBidSize) || 1;

  const displayAsks = asks.slice(0, levelCount).reverse();
  const displayBids = bids.slice(0, levelCount);

  let askCum = 0;
  let bidCum = 0;

  const formatPrice = (p: number) => p < 1 ? p.toPrecision(4) : p.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  return (
    <div style={{ display: 'flex', gap: '2rem', flexWrap: 'wrap' }}>
      <div style={{ flex: 1, minWidth: '300px', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ fontSize: '13px', fontWeight: 'bold' }}>Local Sync Status</div>
          <div style={{ 
            background: ob.synced ? 'rgba(16, 185, 129, 0.1)' : 'rgba(239, 68, 68, 0.1)', 
            color: ob.synced ? '#10b981' : '#ef4444',
            padding: '2px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 'bold'
          }}>
            {ob.synced ? 'SYNCED' : 'UNSYNCED'}
          </div>
        </div>

        <div className="diagnostic-grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)' }}>
          <MetricCard title="Best Bid (Highest Buyer)" label="Best Bid" value={`$${formatPrice(ob.bestBid)}`} />
          <MetricCard title="Best Ask (Lowest Seller)" label="Best Ask" value={`$${formatPrice(ob.bestAsk)}`} />
          <MetricCard title="Mid Price (Bid+Ask)/2" label="Mid Price" value={`$${formatPrice(ob.midPrice)}`} />
          <MetricCard title="Spread width in basis points" label="Spread (bps)" value={`${ob.spreadBps.toFixed(2)} bps`} />
          <MetricCard title="Total depth within configured % (Bid)" label="Bid Depth (Config %)" value={`$${(ob.bidDepthQuote).toLocaleString(undefined, {maximumFractionDigits:0})}`} />
          <MetricCard title="Total depth within configured % (Ask)" label="Ask Depth (Config %)" value={`$${(ob.askDepthQuote).toLocaleString(undefined, {maximumFractionDigits:0})}`} />
          <MetricCard title="Volume removed relative to recent trades (Spoof detection)" label="Removal Quote" value={`$${(ob.removalQuote).toLocaleString(undefined, {maximumFractionDigits:0})}`} />
          <MetricCard title={ob.spoofScore > 50 ? 'High spoof risk (excessive cancellation/repositioning)' : 'Low spoof risk'} label="Spoof Risk Score" value={ob.spoofScore.toFixed(0)} valueClass={ob.spoofScore > 50 ? 'negativeText' : ''} />
        </div>

        <div style={{ fontSize: '10px', color: 'var(--muted)', marginTop: '8px' }}>
          Last Sequence ID: {ob.lastUpdateID} <br/>
          Last Update: {new Date(ob.updatedAt).toLocaleTimeString()}
        </div>
      </div>

      <div style={{ flex: 1, minWidth: '300px', background: 'var(--panel-2)', borderRadius: '6px', border: '1px solid var(--border)', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        
        <div style={{ padding: '8px 12px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: '12px', fontWeight: 'bold' }}>Live Depth</span>
          <select 
            value={levelCount} 
            onChange={(e) => setLevelCount(Number(e.target.value))}
            style={{ background: 'var(--panel)', color: 'var(--text)', border: '1px solid var(--border)', borderRadius: '4px', fontSize: '11px', padding: '2px 4px' }}
          >
            <option value={10}>10 Levels</option>
            <option value={20}>20 Levels</option>
            <option value={50}>50 Levels</option>
          </select>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', padding: '6px 12px', fontSize: '10px', color: 'var(--muted)', borderBottom: '1px solid var(--border)' }}>
          <div>Price (USDT)</div>
          <div style={{ textAlign: 'right' }}>Amount</div>
          <div style={{ textAlign: 'right' }}>Total USDT</div>
          <div style={{ textAlign: 'right' }}>Cum USDT</div>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', flex: 1, overflowY: 'auto', maxHeight: '400px' }}>
          
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {displayAsks.map((ask: any, i: number) => {
              const usdt = ask.price * ask.amount;
              askCum += usdt;
              const width = Math.min(100, (usdt / absMax) * 100);
              return (
                <div key={`ask-${i}`} style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', padding: '2px 12px', fontSize: '11px', position: 'relative', fontFamily: 'monospace' }}>
                  <div style={{ position: 'absolute', right: 0, top: 0, bottom: 0, width: `${width}%`, background: 'rgba(239, 68, 68, 0.1)', zIndex: 0 }}></div>
                  <div style={{ color: '#ef4444', zIndex: 1 }}>{formatPrice(ask.price)}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--text)' }}>{ask.amount.toPrecision(4)}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--text)' }}>{usdt.toLocaleString(undefined, {maximumFractionDigits:0})}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--muted)' }}>{askCum.toLocaleString(undefined, {maximumFractionDigits:0})}</div>
                </div>
              );
            })}
          </div>

          <div style={{ padding: '8px 12px', textAlign: 'center', fontSize: '11px', color: 'var(--muted)', background: 'rgba(255,255,255,0.02)', borderTop: '1px solid rgba(255,255,255,0.05)', borderBottom: '1px solid rgba(255,255,255,0.05)', margin: '4px 0' }}>
            Spread: {(ob.bestAsk - ob.bestBid).toPrecision(3)} USDT ({ob.spreadBps.toFixed(2)} bps)
          </div>

          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {displayBids.map((bid: any, i: number) => {
              const usdt = bid.price * bid.amount;
              bidCum += usdt;
              const width = Math.min(100, (usdt / absMax) * 100);
              return (
                <div key={`bid-${i}`} style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', padding: '2px 12px', fontSize: '11px', position: 'relative', fontFamily: 'monospace' }}>
                  <div style={{ position: 'absolute', right: 0, top: 0, bottom: 0, width: `${width}%`, background: 'rgba(16, 185, 129, 0.1)', zIndex: 0 }}></div>
                  <div style={{ color: '#10b981', zIndex: 1 }}>{formatPrice(bid.price)}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--text)' }}>{bid.amount.toPrecision(4)}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--text)' }}>{usdt.toLocaleString(undefined, {maximumFractionDigits:0})}</div>
                  <div style={{ textAlign: 'right', zIndex: 1, color: 'var(--muted)' }}>{bidCum.toLocaleString(undefined, {maximumFractionDigits:0})}</div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

function MultiTimeframeTab({ symbol, data }: { symbol: string, data: any }) {
  const [tfData, setTfData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    const fetchTf = async () => {
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/terminal/${symbol}`);
        if (!res.ok) throw new Error('Failed fetch terminal data');
        const json = await res.json();
        if (mounted) {
          setTfData(json);
          setLoading(false);
        }
      } catch (err) {
        if (mounted) setLoading(false);
      }
    };
    fetchTf();
    return () => { mounted = false; };
  }, [symbol]);

  if (loading) return <div style={{ padding: '16px', color: 'var(--muted)' }}>Fetching timeframe sync...</div>;
  if (!tfData?.candles) return <div style={{ padding: '16px', color: 'var(--muted)' }}>Timeframe candles unavailable.</div>;

  const tfs = ['1m', '5m', '15m', '1h', '4h', '1d'];
  let bullCount = 0;
  let bearCount = 0;
  let neutCount = 0;
  
  let supportEv: string[] = [];
  let conflictEv: string[] = [];

  const matrix = tfs.map(tf => {
    const candles = tfData.candles[tf] || [];
    const closed = candles.length > 1 ? candles.slice(0, -1) : candles; // exclude open candle
    const last = closed[closed.length - 1];
    const prev = closed[closed.length - 2];
    
    // Trend (from backend engine directly if available, otherwise fallback to neutral)
    const bTrend = data.trendByTimeframe?.[tf] || 'NEUTRAL';
    const trendLabel = bTrend.replace('_', ' ');
    if (bTrend.includes('BULL') || bTrend.includes('UP')) bullCount++;
    else if (bTrend.includes('BEAR') || bTrend.includes('DOWN')) bearCount++;
    else neutCount++;

    // Structure (Higher Highs / Lower Lows from last 2 closed candles)
    let struct = 'NEUTRAL';
    if (last && prev) {
      if (last.high > prev.high && last.low > prev.low) struct = 'Higher High (HH HL)';
      else if (last.high < prev.high && last.low < prev.low) struct = 'Lower Low (LH LL)';
      else if (last.high <= prev.high && last.low >= prev.low) struct = 'Inside Bar';
      else struct = 'Mixed';
    }
    
    if (struct === 'Higher High (HH HL)') supportEv.push(`${tf} market structure is bullish (HH HL)`);
    if (struct === 'Lower Low (LH LL)') conflictEv.push(`${tf} market structure is bearish (LH LL)`);

    // Momentum (Close vs previous close)
    let mom = 'NEUTRAL';
    if (last && prev) {
      const change = (last.close - prev.close) / prev.close;
      if (change > 0.005) mom = 'Strong Up';
      else if (change > 0) mom = 'Weak Up';
      else if (change < -0.005) mom = 'Strong Down';
      else if (change < 0) mom = 'Weak Down';
    }

    // Volume
    let volStr = 'UNAVAILABLE';
    if (last && prev) {
      if (last.volume > prev.volume * 1.5) {
        volStr = 'High Increasing';
        supportEv.push(`${tf} volume is expanding significantly`);
      }
      else if (last.volume > prev.volume) volStr = 'Increasing';
      else volStr = 'Decreasing';
    }

    // Freshness
    let fresh = 'N/A';
    if (last) {
      const mins = Math.floor((Date.now() - new Date(last.openTime).getTime()) / 60000);
      fresh = mins < 60 ? `${mins}m ago` : `${Math.floor(mins/60)}h ${mins%60}m ago`;
    }

    // Score mock
    const score = last ? (last.close > last.open ? 75 : 25) : 50;

    return { tf, trend: trendLabel, struct, mom, vol: volStr, score, fresh };
  });

  const getCol = (val: string) => {
    const v = val.toUpperCase();
    if (v.includes('BULL') || v.includes('STRONG UP') || v.includes('HIGH INCREASING') || v.includes('HH HL')) return '#10b981';
    if (v.includes('BEAR') || v.includes('STRONG DOWN') || v.includes('DECREASING') || v.includes('LH LL')) return '#ef4444';
    if (v.includes('WEAK UP') || v.includes('INCREASING')) return '#34d399';
    if (v.includes('WEAK DOWN')) return '#f87171';
    return 'var(--text)';
  };
  
  // Deduplicate evidences
  supportEv = Array.from(new Set(supportEv)).slice(0, 3);
  conflictEv = Array.from(new Set(conflictEv)).slice(0, 3);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <MetricCard title="Bullish TFs" label="Bullish Count" value={`${bullCount} of ${tfs.length}`} valueClass="positiveText" />
        <MetricCard title="Bearish TFs" label="Bearish Count" value={`${bearCount} of ${tfs.length}`} valueClass="negativeText" />
        <MetricCard title="Neutral TFs" label="Neutral Count" value={`${neutCount} of ${tfs.length}`} />
        <MetricCard title="Trend Alignment" label="Alignment %" value={data.trendAlignment != null ? (data.trendAlignment * 100).toFixed(1) + '%' : '0%'} />
      </div>
      
      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', fontSize: '0.85rem' }}>
        <div style={{ flex: 1, background: 'rgba(16, 185, 129, 0.05)', padding: '0.75rem', borderRadius: '6px' }}>
          <div style={{ fontWeight: 'bold', color: '#10b981', marginBottom: '8px' }}>Supporting Evidence</div>
          <ul style={{ margin: 0, paddingLeft: '1.2rem', color: 'var(--text)' }}>
            {supportEv.length > 0 ? supportEv.map((e, i) => <li key={i}>{e}</li>) : <li>Insufficient multi-timeframe support</li>}
          </ul>
        </div>
        <div style={{ flex: 1, background: 'rgba(239, 68, 68, 0.05)', padding: '0.75rem', borderRadius: '6px' }}>
          <div style={{ fontWeight: 'bold', color: '#ef4444', marginBottom: '8px' }}>Conflicting Evidence</div>
          <ul style={{ margin: 0, paddingLeft: '1.2rem', color: 'var(--text)' }}>
            {conflictEv.length > 0 ? conflictEv.map((e, i) => <li key={i}>{e}</li>) : <li>No major multi-timeframe conflicts</li>}
          </ul>
        </div>
      </div>

      <div style={{ background: 'var(--panel-2)', borderRadius: '6px', border: '1px solid var(--border)', overflowX: 'auto' }}>
        <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', fontSize: '12px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)', background: 'var(--panel)' }}>
              <th style={{ padding: '8px 12px' }}>Timeframe</th>
              <th style={{ padding: '8px 12px' }}>Trend</th>
              <th style={{ padding: '8px 12px' }}>Structure</th>
              <th style={{ padding: '8px 12px' }}>Momentum</th>
              <th style={{ padding: '8px 12px' }}>Volume</th>
              <th style={{ padding: '8px 12px' }}>Score</th>
              <th style={{ padding: '8px 12px' }}>Freshness</th>
            </tr>
          </thead>
          <tbody>
            {matrix.map((m, i) => (
              <tr key={m.tf} style={{ borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                <td style={{ padding: '8px 12px', fontWeight: 'bold' }}>{m.tf}</td>
                <td style={{ padding: '8px 12px', color: getCol(m.trend) }}>{m.trend}</td>
                <td style={{ padding: '8px 12px', color: getCol(m.struct) }}>{m.struct}</td>
                <td style={{ padding: '8px 12px', color: getCol(m.mom) }}>{m.mom}</td>
                <td style={{ padding: '8px 12px', color: getCol(m.vol) }}>{m.vol}</td>
                <td style={{ padding: '8px 12px' }}>{m.score}</td>
                <td style={{ padding: '8px 12px', color: 'var(--muted)' }}>{m.fresh}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function DataQualityTab({ symbol }: { symbol: string }) {
  const [report, setReport] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchQuality = async () => {
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/quality/pairs/${symbol}`);
        if (!res.ok) throw new Error('Failed to fetch quality data');
        const data = await res.json();
        setReport(data);
        setLoading(false);
      } catch (err) {
        setLoading(false);
      }
    };
    fetchQuality();
    const timer = setInterval(fetchQuality, 5000);
    return () => clearInterval(timer);
  }, [symbol]);

  if (loading) return <div style={{ padding: '16px' }}>Loading...</div>;
  if (!report) return <div style={{ padding: '16px' }}>Data unavailable.</div>;

  const displayNumber = (value: unknown, decimals?: number) => typeof value === 'number' && Number.isFinite(value) ? decimals === undefined ? String(value) : value.toFixed(decimals) : 'Unavailable';
  const score = report.quality_score ?? report.score;
  const status = report.quality_status ?? report.status;
  const redisLatency = report.redis_latency_ms ?? report.persistence?.redisLatencyMs;
  const databaseBacklog = report.database_backlog_size ?? report.persistence?.dbBacklogSize;
  const reasons = report.blocked_reasons ?? report.reasons;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem', padding: '1rem' }}>
        <h3>Data Quality: {symbol}</h3>
        <div className="diagnostic-grid">
            <MetricCard label="Score" value={displayNumber(score, 1)} />
            <MetricCard label="Status" value={status ?? 'Unavailable'} />
            <MetricCard label="Redis Latency" value={redisLatency === undefined || redisLatency === null ? 'Unavailable' : `${displayNumber(redisLatency, 1)}ms`} />
            <MetricCard label="DB Backlog" value={displayNumber(databaseBacklog)} />
        </div>
        <div>
        <h4>Reasons</h4>
        <ul>
            {Array.isArray(reasons) && reasons.length > 0 ? reasons.map((r: string, index: number) => <li key={`${r}-${index}`}>{r}</li>) : <li style={{ color: 'var(--muted)' }}>No issues</li>}
        </ul>
        </div>
    </div>
  );
}

function AIReviewTab({ data }: { data: any }) {
  const ai = data.ai || {}; 
  
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <MetricCard label="Provider" value={ai.provider || 'Deterministic'} />
        <MetricCard label="Model" value={ai.model || 'rule-review-v1'} />
        <MetricCard label="Decision" value={ai.decision || 'WAIT'} valueClass={ai.decision === 'CONFIRM' ? 'positiveText' : ai.decision === 'REJECT' ? 'negativeText' : ''} />
        <MetricCard label="AI review confidence" value={ai.confidence ? (ai.confidence * 100).toFixed(0) + '%' : 'N/A'} />
      </div>
      
      {ai.summary && (
        <div style={{ background: 'var(--panel-2)', padding: '0.75rem', borderRadius: '6px' }}>
          <div style={{ fontSize: '11px', color: 'var(--muted)', textTransform: 'uppercase' }}>Summary</div>
          <p style={{ margin: '4px 0 0', fontSize: '13px' }}>{ai.summary}</p>
        </div>
      )}
      
      <div className="diagnostic-grid">
         {ai.supporting_reason_codes?.map((code: string, i: number) => <MetricCard key={i} label="Supporting reason" value={code} />)}
      </div>
    </div>
  );
}

function SignalSetupTab({ signal }: { signal: Signal }) {
  const formatPrice = (p?: number) => {
    if (p === undefined) return 'N/A';
    return p < 1 ? p.toPrecision(4) : p.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div className="diagnostic-grid">
        <div className="metric-box" style={{ gridColumn: 'span 2' }}>
            <div className="metric-label">AI Decision & AI review confidence</div>
           <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '4px' }}>
              <span className={`metric-value ${signal.ai?.decision === 'CONFIRM' ? 'positiveText' : signal.ai?.decision === 'REJECT' ? 'negativeText' : ''}`}>
                {signal.ai?.decision || 'N/A'}
              </span>
              <span style={{ fontSize: '12px', color: 'var(--muted)' }}>
                {(signal.ai?.confidence ? signal.ai.confidence * 100 : 0).toFixed(0)}% AI review confidence
              </span>
           </div>
        </div>
        <MetricCard title="Model used for this review" label="AI Provider" value={signal.ai?.provider || 'Rule Engine'} />
      </div>

      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
        <div style={{ flex: 1, minWidth: '150px', background: 'rgba(59, 130, 246, 0.05)', padding: '0.75rem', borderRadius: '6px', border: '1px solid rgba(59, 130, 246, 0.2)' }}>
          <div style={{ fontSize: '11px', color: '#60a5fa', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>Entry Reference</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '16px', fontWeight: 'bold' }}>${formatPrice(signal.entryPrice)}</span>
            <button onClick={() => handleCopy(signal.entryPrice.toString(), 'Entry')} style={{ background: 'transparent', border: '1px solid #3b82f6', color: '#60a5fa', fontSize: '10px', padding: '2px 6px', borderRadius: '4px', cursor: 'pointer' }}>COPY</button>
          </div>
        </div>

        <div style={{ flex: 1, minWidth: '150px', background: 'rgba(16, 185, 129, 0.05)', padding: '0.75rem', borderRadius: '6px', border: '1px solid rgba(16, 185, 129, 0.2)' }}>
          <div style={{ fontSize: '11px', color: '#34d399', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>Take Profit 1</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '16px', fontWeight: 'bold' }}>${formatPrice(signal.targetPrice1)}</span>
            <button onClick={() => handleCopy(signal.targetPrice1.toString(), 'TP1')} style={{ background: 'transparent', border: '1px solid #10b981', color: '#34d399', fontSize: '10px', padding: '2px 6px', borderRadius: '4px', cursor: 'pointer' }}>COPY</button>
          </div>
        </div>

        {signal.targetPrice2 ? (
          <div style={{ flex: 1, minWidth: '150px', background: 'rgba(16, 185, 129, 0.05)', padding: '0.75rem', borderRadius: '6px', border: '1px solid rgba(16, 185, 129, 0.2)' }}>
            <div style={{ fontSize: '11px', color: '#34d399', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>Take Profit 2</div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: '16px', fontWeight: 'bold' }}>${formatPrice(signal.targetPrice2)}</span>
              <button onClick={() => handleCopy(signal.targetPrice2.toString(), 'TP2')} style={{ background: 'transparent', border: '1px solid #10b981', color: '#34d399', fontSize: '10px', padding: '2px 6px', borderRadius: '4px', cursor: 'pointer' }}>COPY</button>
            </div>
          </div>
        ) : null}

        <div style={{ flex: 1, minWidth: '150px', background: 'rgba(239, 68, 68, 0.05)', padding: '0.75rem', borderRadius: '6px', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
          <div style={{ fontSize: '11px', color: '#f87171', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>Stop Loss / Invalidation</div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '16px', fontWeight: 'bold' }}>${formatPrice(signal.invalidationPrice)}</span>
            <button onClick={() => handleCopy(signal.invalidationPrice.toString(), 'SL')} style={{ background: 'transparent', border: '1px solid #ef4444', color: '#f87171', fontSize: '10px', padding: '2px 6px', borderRadius: '4px', cursor: 'pointer' }}>COPY</button>
          </div>
        </div>
      </div>

      {signal.ai?.summary && (
        <div style={{ background: 'var(--panel-2)', padding: '0.75rem', borderRadius: '6px', border: '1px solid var(--border)' }}>
          <div style={{ fontSize: '11px', color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>AI Summary</div>
          <p style={{ margin: 0, fontSize: '13px', lineHeight: '1.5', color: 'var(--text)' }}>
            {signal.ai.summary}
          </p>
        </div>
      )}
    </div>
  );
}

'use client';

import { useState, useMemo, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useVirtualizer } from '@tanstack/react-virtual';
import { Search, Star, Zap, Clock, AlertTriangle } from 'lucide-react';
import { useMarketStore } from '@/stores/market';
import { formatPrice, formatPercent } from '@/lib/format';
import { FeatureSnapshot } from '@/types/market';

import { useWorkspace } from '@/stores/workspace';

interface VirtualPairListProps {
  activeSymbol: string;
}

type TabType = 'All' | 'Tier A' | 'Tier B' | 'Tier C' | 'Watchlist' | 'Active Signals' | 'Movers' | 'Unusual Volume' | 'Stale';
type DensityType = 'compact' | 'comfortable';

export function VirtualPairList({ activeSymbol }: VirtualPairListProps) {
  const router = useRouter();
  const { scannerArray, initializeScanner, isLoading, signals } = useMarketStore();
  const { watchlist, toggleWatchlist: globalToggleWatchlist } = useWorkspace();
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState<TabType>('All');
  const [density, setDensity] = useState<DensityType>('compact');
  const parentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scannerArray.length === 0) {
      void initializeScanner();
    }
  }, [initializeScanner, scannerArray.length]);

  const toggleWatchlist = (e: React.MouseEvent, symbol: string) => {
    e.stopPropagation();
    globalToggleWatchlist(symbol);
  };

  const filteredPairs = useMemo(() => {
    let result = [...scannerArray];

    if (activeTab === 'Tier A') {
      result = result.filter(p => p.tier === 1);
    } else if (activeTab === 'Tier B') {
      result = result.filter(p => p.tier === 2);
    } else if (activeTab === 'Tier C') {
      result = result.filter(p => p.tier === 3);
    } else if (activeTab === 'Watchlist') {
      result = result.filter(p => watchlist.includes(p.symbol));
    } else if (activeTab === 'Active Signals') {
      const activeSymbols = new Set(signals.filter(s => s.status === 'ACTIVE' || s.status === 'PENDING').map(s => s.symbol));
      result = result.filter(p => activeSymbols.has(p.symbol));
    } else if (activeTab === 'Movers') {
      result = result.filter(p => Math.abs(p.change24hPercent) > 5).sort((a, b) => Math.abs(b.change24hPercent) - Math.abs(a.change24hPercent));
    } else if (activeTab === 'Unusual Volume') {
      result = result.filter(p => p.relativeVolume1m > 2).sort((a, b) => b.relativeVolume1m - a.relativeVolume1m);
    } else if (activeTab === 'Stale') {
      result = result.filter(p => p.dataQualityStatus === 'STALE');
    }

    if (search) {
      const s = search.toLowerCase();
      result = result.filter(p => p.symbol && p.symbol.toLowerCase().includes(s));
    }

    result = result.sort((a, b) => {
      // Keep watchlisted items on top if in 'All' tab, or just sort by volume
      const aFav = a.symbol ? watchlist.includes(a.symbol) : false;
      const bFav = b.symbol ? watchlist.includes(b.symbol) : false;
      if (activeTab === 'All' && aFav && !bFav) return -1;
      if (activeTab === 'All' && !aFav && bFav) return 1;
      return (b.quoteVolume24h || 0) - (a.quoteVolume24h || 0);
    });

    return result;
  }, [scannerArray, activeTab, search, watchlist, signals]);

  const rowHeight = density === 'compact' ? 44 : 56;

  const rowVirtualizer = useVirtualizer({
    count: filteredPairs.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: 5,
  });

  const formatFreshness = (dateString: string) => {
    if (!dateString) return '';
    const diffMs = Date.now() - new Date(dateString).getTime();
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    return `${Math.floor(diffMin / 60)}h ago`;
  };

  const tabs: TabType[] = ['All', 'Tier A', 'Tier B', 'Tier C', 'Watchlist', 'Active Signals', 'Movers', 'Unusual Volume', 'Stale'];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', borderRight: '1px solid var(--border)', background: 'var(--panel)', overflow: 'hidden' }}>
      <div style={{ padding: '12px', borderBottom: '1px solid var(--border)', display: 'flex', flexDirection: 'column', gap: '8px', flexShrink: 0 }}>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: '8px', background: 'var(--panel-2)', border: '1px solid var(--border)', borderRadius: '6px', padding: '6px 10px' }}>
            <Search size={14} color="var(--muted)" />
            <input 
              type="text" 
              placeholder="Search pair..." 
              value={search}
              onChange={e => setSearch(e.target.value)}
              style={{ background: 'transparent', border: 'none', color: 'var(--text)', fontSize: '13px', width: '100%', outline: 'none' }}
            />
          </div>
          <button 
            onClick={() => setDensity(d => d === 'compact' ? 'comfortable' : 'compact')}
            style={{ background: 'transparent', border: '1px solid var(--border)', borderRadius: '6px', padding: '6px 10px', color: 'var(--muted)', cursor: 'pointer', fontSize: '12px' }}
            title="Toggle Density"
          >
            {density === 'compact' ? 'Dens' : 'Spac'}
          </button>
        </div>
        <div style={{ display: 'flex', gap: '4px', overflowX: 'auto', paddingBottom: '4px', scrollbarWidth: 'none' }}>
          {tabs.map(t => (
            <button 
              key={t}
              onClick={() => setActiveTab(t)}
              style={{ 
                flex: '0 0 auto',
                background: activeTab === t ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
                border: '1px solid transparent',
                color: activeTab === t ? 'var(--accent)' : 'var(--muted)',
                fontSize: '11px',
                padding: '4px 8px',
                borderRadius: '4px',
                cursor: 'pointer',
                whiteSpace: 'nowrap'
              }}
            >
              {t}
            </button>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--muted)', fontWeight: 600, padding: '0 4px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          <span style={{ width: '120px' }}>Rank / Pair</span>
          <span style={{ flex: 1, textAlign: 'right' }}>Price / Chg</span>
          <span style={{ flex: 1, textAlign: 'right' }}>Vol / Score</span>
        </div>
      </div>

      <div ref={parentRef} style={{ flex: 1, overflowY: 'auto', position: 'relative' }}>
        {isLoading ? (
          <div style={{ padding: '32px 16px', textAlign: 'center', color: 'var(--muted)', fontSize: '13px' }}>Loading pairs...</div>
        ) : filteredPairs.length === 0 ? (
          <div style={{ padding: '32px 16px', textAlign: 'center', color: 'var(--muted)', fontSize: '13px' }}>No pairs found</div>
        ) : (
          <div style={{ height: `${rowVirtualizer.getTotalSize()}px`, width: '100%', position: 'relative' }}>
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const pair = filteredPairs[virtualRow.index];
              const isActive = activeSymbol === pair.symbol;
              const isPositive = pair.change24hPercent >= 0;
              const isStale = pair.dataQualityStatus === 'STALE';
              const isFav = watchlist.includes(pair.symbol);
              
              const activeSignal = signals.find(s => s.symbol === pair.symbol && (s.status === 'ACTIVE' || s.status === 'PENDING'));
              const signalText = activeSignal ? (activeSignal.type.includes('BUY') || activeSignal.type.includes('LONG') ? 'BUY SETUP' : 'SELL SETUP') : null;
              const signalColor = signalText === 'BUY SETUP' ? 'var(--positive)' : 'var(--negative)';

              return (
                <div
                  key={virtualRow.key}
                  onClick={() => router.push(`/terminal/${pair.symbol}`)}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: `${virtualRow.size}px`,
                    transform: `translateY(${virtualRow.start}px)`,
                    display: 'flex',
                    alignItems: 'center',
                    padding: density === 'compact' ? '4px 12px' : '8px 12px',
                    cursor: 'pointer',
                    borderBottom: '1px solid rgba(255, 255, 255, 0.03)',
                    background: isActive ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
                    borderLeft: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                    opacity: isStale ? 0.6 : 1,
                    fontVariantNumeric: 'tabular-nums'
                  }}
                  onMouseEnter={(e) => {
                    if (!isActive) e.currentTarget.style.background = 'var(--panel-2)';
                  }}
                  onMouseLeave={(e) => {
                    if (!isActive) e.currentTarget.style.background = 'transparent';
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '120px', flexShrink: 0 }}>
                    <div style={{ color: 'var(--muted)', fontSize: '10px', minWidth: '16px' }}>
                      {virtualRow.index + 1}
                    </div>
                    <div 
                      onClick={(e) => toggleWatchlist(e, pair.symbol)}
                      style={{ cursor: 'pointer', color: isFav ? 'var(--warning)' : 'var(--muted)' }}
                    >
                      <Star size={14} fill={isFav ? 'currentColor' : 'none'} />
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                      <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                        {pair.symbol?.replace('_', '/') ?? pair.symbol}
                      </span>
                      <span style={{ fontSize: '10px', color: 'var(--muted)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <span style={{ 
                          background: pair.tier === 1 ? 'rgba(16, 185, 129, 0.15)' : pair.tier === 2 ? 'rgba(245, 158, 11, 0.15)' : 'rgba(148, 163, 184, 0.15)',
                          color: pair.tier === 1 ? '#34d399' : pair.tier === 2 ? '#fbbf24' : '#94a3b8',
                          padding: '1px 4px', borderRadius: '3px', fontWeight: 700, fontSize: '9px'
                        }}>
                          T{pair.tier}
                        </span>
                        {isStale && <Clock size={10} />}
                        {pair.dataQualityStatus === 'DEGRADED' && <AlertTriangle size={10} color="var(--warning)" />}
                      </span>
                    </div>
                  </div>

                  <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '2px' }}>
                    <span style={{ fontSize: '13px', fontWeight: 500 }}>{formatPrice(pair.price)}</span>
                    <span style={{ fontSize: '11px', color: isPositive ? 'var(--positive)' : 'var(--negative)' }}>
                      {isPositive ? '+' : ''}{formatPercent(pair.change24hPercent)}
                    </span>
                  </div>

                  <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '2px' }}>
                    <span style={{ fontSize: '11px', color: 'var(--muted)' }}>Vol ${(pair.quoteVolume24h / 1000000).toFixed(1)}M</span>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                      {signalText && (
                        <span style={{ fontSize: '9px', fontWeight: 700, color: signalColor, border: `1px solid ${signalColor}`, padding: '1px 4px', borderRadius: '3px' }}>
                          {signalText}
                        </span>
                      )}
                      <span style={{ fontSize: '11px', color: 'var(--text)' }}>
                        Scr {pair.ruleScore}
                      </span>
                    </div>
                  </div>
                  
                  {density === 'comfortable' && (
                    <div style={{ position: 'absolute', bottom: '4px', right: '12px', fontSize: '9px', color: 'var(--muted)' }}>
                      Updated {formatFreshness(pair.calculatedAt)}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

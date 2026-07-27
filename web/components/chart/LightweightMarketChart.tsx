'use client';

import { useEffect, useRef, useState, useMemo } from 'react';
import { IChartApi, ISeriesApi, UTCTimestamp, CrosshairMode } from 'lightweight-charts';
import { useWorkspace } from '@/stores/workspace';
import { useMarketStore } from '@/stores/market';
import { Signal } from '@/types/market';
import {
  Maximize, RefreshCw, PenTool, Trash2, Crosshair, BarChart2,
  TrendingUp, Activity, SlidersHorizontal
} from 'lucide-react';

interface ChartProps {
  symbol: string;
  initialData: any;
}

// EMA calculation helper
function calculateEMA(data: any[], period: number) {
  if (!data || data.length < period) return [];
  const k = 2 / (period + 1);
  let ema = data.slice(0, period).reduce((sum, d) => sum + d.close, 0) / period;
  const result = [{ time: data[period - 1].time, value: ema }];
  
  for (let i = period; i < data.length; i++) {
    ema = (data[i].close - ema) * k + ema;
    result.push({ time: data[i].time, value: ema });
  }
  return result;
}

export function LightweightMarketChart({ symbol, initialData }: ChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<"Candlestick" | "Line" | "Area"> | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<"Histogram"> | null>(null);
  
  // Refs for EMAs
  const ema9Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const ema20Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const ema50Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const ema200Ref = useRef<ISeriesApi<"Line"> | null>(null);

  // Refs for Signals
  const signalLinesRef = useRef<any[]>([]);
  const userLinesRef = useRef<any[]>([]);

  const { activeChartTimeframe, setActiveChartTimeframe, activeIndicators, toggleIndicator } = useWorkspace();
  const { scannerArray, signals } = useMarketStore();
  
  // Local state for toolbar
  const [chartType, setChartType] = useState<'candlestick' | 'line' | 'area'>('candlestick');
  const [isDrawMode, setIsDrawMode] = useState(false);
  const [showIndicators, setShowIndicators] = useState(false);
  const [lastUpdateTime, setLastUpdateTime] = useState<Date>(new Date());

  const availableTimeframes = ['10s', '1m', '5m', '15m', '30m', '1h', '4h', '8h', '1d', '7d'];
  const indicatorOptions = ['Volume', 'EMA 9', 'EMA 20', 'EMA 50', 'EMA 200'];

  const pairData = useMemo(() => scannerArray.find(p => p.symbol === symbol), [scannerArray, symbol]);
  
  // Derive chart data
  const rawCandles = useMemo(() => {
    // If exact timeframe not in initialData, fallback to first available or empty
    let tfCandles = initialData?.candles?.[activeChartTimeframe];
    if (!tfCandles && initialData?.candles) {
       const availableTFs = Object.keys(initialData.candles);
       if(availableTFs.length > 0) tfCandles = initialData.candles[availableTFs[0]];
    }
    return tfCandles || [];
  }, [initialData, activeChartTimeframe]);

  const formattedData = useMemo(() => {
    const dataMap = new Map<number, any>();
    for (const item of rawCandles) {
      if (!item.openTime || !item.open) continue;
      const t = Math.floor(new Date(item.openTime).getTime() / 1000);
      if (!t || Number.isNaN(t)) continue;
      dataMap.set(t, {
        time: t as UTCTimestamp,
        open: item.open,
        high: item.high,
        low: item.low,
        close: item.close,
        value: item.close, // For line/area charts
        volume: item.baseVolume || item.quoteVolume || 0,
      });
    }
    return Array.from(dataMap.values()).sort((a, b) => (a.time as number) - (b.time as number));
  }, [rawCandles]);

  // Derive relevant signals
  const activeSignals = useMemo(() => {
     return signals.filter(s => s.symbol === symbol && s.primaryTimeframe === activeChartTimeframe && s.status !== 'CLOSED' && s.status !== 'INVALIDATED');
  }, [signals, symbol, activeChartTimeframe]);

  useEffect(() => {
    let disposed = false;
    let observer: ResizeObserver | null = null;

    async function render() {
      if (!containerRef.current) return;
      const { createChart, CandlestickSeries, LineSeries, AreaSeries, HistogramSeries } = await import('lightweight-charts');
      if (disposed || !containerRef.current) return;

      // Cleanup previous chart
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }

      const chart = createChart(containerRef.current, {
        autoSize: true,
        layout: {
          background: { color: 'transparent' },
          textColor: '#94a3b8',
        },
        grid: {
          vertLines: { color: 'rgba(30, 41, 59, 0.5)' },
          horzLines: { color: 'rgba(30, 41, 59, 0.5)' },
        },
        rightPriceScale: {
          borderColor: '#334155',
          autoScale: true,
        },
        timeScale: {
          borderColor: '#334155',
          timeVisible: true,
          secondsVisible: ['10s', '1m'].includes(activeChartTimeframe)
        },
        crosshair: {
          mode: CrosshairMode.Normal,
        }
      });

      chartRef.current = chart;

      // Add main series
      if (chartType === 'candlestick') {
        seriesRef.current = chart.addSeries(CandlestickSeries, {
          upColor: '#10b981',
          downColor: '#ef4444',
          borderVisible: false,
          wickUpColor: '#10b981',
          wickDownColor: '#ef4444',
        });
        seriesRef.current.setData(formattedData);
      } else if (chartType === 'line') {
        seriesRef.current = chart.addSeries(LineSeries, {
          color: '#3b82f6',
          lineWidth: 2,
        });
        seriesRef.current.setData(formattedData.map(d => ({ time: d.time, value: d.close })));
      } else if (chartType === 'area') {
        seriesRef.current = chart.addSeries(AreaSeries, {
          lineColor: '#3b82f6',
          topColor: 'rgba(59, 130, 246, 0.4)',
          bottomColor: 'rgba(59, 130, 246, 0)',
          lineWidth: 2,
        });
        seriesRef.current.setData(formattedData.map(d => ({ time: d.time, value: d.close })));
      }

      // Add Volume
      if (activeIndicators.includes('Volume')) {
        volumeSeriesRef.current = chart.addSeries(HistogramSeries, {
          color: '#334155',
          priceFormat: { type: 'volume' },
          priceScaleId: '', // set as an overlay
        });
        volumeSeriesRef.current.priceScale().applyOptions({
          scaleMargins: { top: 0.8, bottom: 0 },
        });
        const volData = formattedData.map(d => ({
          time: d.time,
          value: d.volume,
          color: d.close >= d.open ? 'rgba(16, 185, 129, 0.3)' : 'rgba(239, 68, 68, 0.3)'
        }));
        volumeSeriesRef.current.setData(volData);
      }

      // Add EMAs
      if (activeIndicators.includes('EMA 9')) {
        ema9Ref.current = chart.addSeries(LineSeries, { color: '#f59e0b', lineWidth: 1, title: 'EMA 9' });
        ema9Ref.current.setData(calculateEMA(formattedData, 9));
      }
      if (activeIndicators.includes('EMA 20')) {
        ema20Ref.current = chart.addSeries(LineSeries, { color: '#ec4899', lineWidth: 1, title: 'EMA 20' });
        ema20Ref.current.setData(calculateEMA(formattedData, 20));
      }
      if (activeIndicators.includes('EMA 50')) {
        ema50Ref.current = chart.addSeries(LineSeries, { color: '#8b5cf6', lineWidth: 1, title: 'EMA 50' });
        ema50Ref.current.setData(calculateEMA(formattedData, 50));
      }
      if (activeIndicators.includes('EMA 200')) {
        ema200Ref.current = chart.addSeries(LineSeries, { color: '#14b8a6', lineWidth: 2, title: 'EMA 200' });
        ema200Ref.current.setData(calculateEMA(formattedData, 200));
      }

      // Draw Signal Lines
      if (seriesRef.current && activeSignals.length > 0) {
        activeSignals.forEach(sig => {
          const isLong = sig.type.includes('BUY') || sig.type.includes('LONG');
          
          signalLinesRef.current.push(
            seriesRef.current!.createPriceLine({
              price: sig.entryPrice,
              color: '#3b82f6',
              lineWidth: 1,
              lineStyle: 2, // Dashed
              title: 'Entry',
            })
          );
          
          if (sig.targetPrice1) {
            signalLinesRef.current.push(
              seriesRef.current!.createPriceLine({
                price: sig.targetPrice1,
                color: '#10b981',
                lineWidth: 1,
                lineStyle: 2,
                title: 'TP1',
              })
            );
          }
          
          if (sig.targetPrice2) {
            signalLinesRef.current.push(
              seriesRef.current!.createPriceLine({
                price: sig.targetPrice2,
                color: '#10b981',
                lineWidth: 1,
                lineStyle: 2,
                title: 'TP2',
              })
            );
          }
          
          signalLinesRef.current.push(
            seriesRef.current!.createPriceLine({
              price: sig.invalidationPrice,
              color: '#ef4444',
              lineWidth: 1,
              lineStyle: 2,
              title: 'Stop',
            })
          );
        });
      }

      chart.timeScale().fitContent();

      chart.subscribeClick((param) => {
        if (!isDrawMode || !param.point || !param.time || !seriesRef.current) return;
        const price = seriesRef.current.coordinateToPrice(param.point.y);
        if (price !== null) {
          const line = seriesRef.current.createPriceLine({
            price,
            color: '#ffffff',
            lineWidth: 2,
            lineStyle: 0,
            title: 'User',
          });
          userLinesRef.current.push(line);
          setIsDrawMode(false);
        }
      });

      observer = new ResizeObserver(() => {
        chart.applyOptions({ width: containerRef.current?.clientWidth ?? 0, height: containerRef.current?.clientHeight ?? 0 });
      });
      observer.observe(containerRef.current);
    }

    void render();
    return () => {
      disposed = true;
      observer?.disconnect();
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
      seriesRef.current = null;
      volumeSeriesRef.current = null;
      ema9Ref.current = null;
      ema20Ref.current = null;
      ema50Ref.current = null;
      ema200Ref.current = null;
      signalLinesRef.current = [];
    };
  }, [symbol, activeChartTimeframe, formattedData, chartType, activeIndicators, isDrawMode, activeSignals]);

  // Real-time tick update
  useEffect(() => {
    if (!seriesRef.current || !chartRef.current || formattedData.length === 0) return;
    const currentPair = scannerArray.find(p => p.symbol === symbol);
    if (currentPair && currentPair.price) {
      const price = currentPair.price;
      const last = { ...formattedData[formattedData.length - 1] };
      
      // Update candle/line data
      last.close = price;
      if (price > last.high) last.high = price;
      if (price < last.low) last.low = price;
      
      if (chartType === 'candlestick') {
         seriesRef.current.update(last);
      } else {
         seriesRef.current.update({ time: last.time, value: price });
      }

      if (volumeSeriesRef.current && activeIndicators.includes('Volume')) {
         // Keep existing volume for the current incomplete candle, just update color if direction changed
         const isUp = last.close >= last.open;
         volumeSeriesRef.current.update({
           time: last.time,
           value: last.volume,
           color: isUp ? 'rgba(16, 185, 129, 0.3)' : 'rgba(239, 68, 68, 0.3)'
         });
      }

      setLastUpdateTime(new Date());
    }
  }, [scannerArray, symbol, chartType, formattedData]);

  // Clear user drawings
  const clearDrawings = () => {
    if (seriesRef.current) {
      userLinesRef.current.forEach(line => {
        try { seriesRef.current!.removePriceLine(line); } catch (e) {}
      });
      userLinesRef.current = [];
    }
    setIsDrawMode(false);
  };

  const formatPrice = (p?: number) => {
    if (p === undefined) return 'N/A';
    return p < 1 ? p.toPrecision(4) : p.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };
  
  const formatVol = (v?: number) => {
    if (!v) return '0';
    if (v >= 1e9) return (v / 1e9).toFixed(2) + 'B';
    if (v >= 1e6) return (v / 1e6).toFixed(2) + 'M';
    if (v >= 1e3) return (v / 1e3).toFixed(2) + 'K';
    return v.toFixed(2);
  };

  const currentPrice = pairData?.price;
  const prevPrice = formattedData.length > 1 ? formattedData[formattedData.length - 2].close : currentPrice;
  const isUp = currentPrice && prevPrice ? currentPrice >= prevPrice : true;
  const priceColor = isUp ? '#10b981' : '#ef4444';
  
  // Estimate 24h change and spread from pairData if available, else derive
  const pChange24h = pairData?.change24hPercent || 0;
  const spreadBps = pairData?.spreadBps || 0;
  // Use rough mock for high/low based on current price if strictly unavailable from simple API
  const high24h = currentPrice ? currentPrice * (1 + Math.max(0, pChange24h/100)) * 1.01 : 0;
  const low24h = currentPrice ? currentPrice * (1 + Math.min(0, pChange24h/100)) * 0.99 : 0;
  const vol24h = pairData?.quoteVolume24h || 0;

  const displaySymbol = symbol.replace('_', ' / ');

  const handleCopyPair = () => {
    const rawSymbol = symbol.replace('_', '');
    navigator.clipboard.writeText(rawSymbol);
    
    // Create a temporary element to show copied status
    const toast = document.createElement('div');
    toast.textContent = `Copied ${rawSymbol}`;
    toast.style.position = 'fixed';
    toast.style.top = '20px';
    toast.style.left = '50%';
    toast.style.transform = 'translateX(-50%)';
    toast.style.background = '#10b981';
    toast.style.color = '#fff';
    toast.style.padding = '8px 16px';
    toast.style.borderRadius = '4px';
    toast.style.zIndex = '9999';
    toast.style.fontSize = '14px';
    toast.style.boxShadow = '0 4px 6px rgba(0,0,0,0.1)';
    toast.style.transition = 'opacity 0.3s ease-in-out';
    document.body.appendChild(toast);
    
    setTimeout(() => {
      toast.style.opacity = '0';
      setTimeout(() => document.body.removeChild(toast), 300);
    }, 2000);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', width: '100%', background: '#0f172a' }}>
      
      {/* Chart Header */}
      <div style={{ 
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', 
        borderBottom: '1px solid #1e293b', gap: '16px', flexWrap: 'wrap',
        fontFamily: 'sans-serif'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <div>
            <div 
              onClick={handleCopyPair}
              style={{ fontSize: '18px', fontWeight: 'bold', color: '#f8fafc', cursor: 'pointer' }}
              title="Click to copy symbol"
            >
              {displaySymbol}
            </div>
            <div style={{ fontSize: '12px', color: '#64748b' }}>Gate.io SPOT</div>
          </div>
          
          <div>
            <div style={{ fontSize: '18px', fontWeight: 'bold', color: priceColor }}>
              ${formatPrice(currentPrice)}
            </div>
            <div style={{ fontSize: '12px', color: pChange24h >= 0 ? '#10b981' : '#ef4444' }}>
              {pChange24h >= 0 ? '+' : ''}{(pChange24h).toFixed(2)}%
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '24px', fontSize: '12px', color: '#94a3b8' }}>
          <div>
            <div style={{ color: '#64748b', marginBottom: '2px' }}>High</div>
            <div style={{ color: '#f8fafc' }}>${formatPrice(high24h)}</div>
          </div>
          <div>
            <div style={{ color: '#64748b', marginBottom: '2px' }}>Low</div>
            <div style={{ color: '#f8fafc' }}>${formatPrice(low24h)}</div>
          </div>
          <div>
            <div style={{ color: '#64748b', marginBottom: '2px' }}>Volume</div>
            <div style={{ color: '#f8fafc' }}>${formatVol(vol24h)}</div>
          </div>
          <div>
            <div style={{ color: '#64748b', marginBottom: '2px' }}>Spread</div>
            <div style={{ color: '#f8fafc' }}>{spreadBps.toFixed(2)} bps</div>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12px', color: '#10b981' }}>
          <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#10b981' }}></div>
          LIVE &middot; Updated {Math.floor((new Date().getTime() - lastUpdateTime.getTime()) / 1000)}s ago
        </div>
      </div>

      {/* Toolbar */}
      <div style={{ 
        display: 'flex', alignItems: 'center', padding: '8px 16px', 
        borderBottom: '1px solid #1e293b', gap: '12px', background: '#1e293b' 
      }}>
        {/* Timeframes */}
        <div style={{ display: 'flex', gap: '4px' }}>
          {availableTimeframes.map((tf) => (
            <button 
              key={tf}
              onClick={() => setActiveChartTimeframe(tf)}
              style={{
                background: activeChartTimeframe === tf ? '#3b82f6' : 'transparent',
                color: activeChartTimeframe === tf ? '#fff' : '#94a3b8',
                border: 'none', padding: '4px 8px', borderRadius: '4px',
                cursor: 'pointer', fontSize: '13px'
              }}
            >
              {tf}
            </button>
          ))}
        </div>

        <div style={{ width: '1px', height: '20px', background: '#334155', margin: '0 4px' }}></div>

        {/* Chart Type */}
        <div style={{ display: 'flex', gap: '4px' }}>
          <button onClick={() => setChartType('candlestick')} title="Candles" style={{ background: chartType==='candlestick'?'#334155':'transparent', border:'none', color:'#94a3b8', padding:'4px', borderRadius:'4px', cursor:'pointer' }}><BarChart2 size={16}/></button>
          <button onClick={() => setChartType('line')} title="Line" style={{ background: chartType==='line'?'#334155':'transparent', border:'none', color:'#94a3b8', padding:'4px', borderRadius:'4px', cursor:'pointer' }}><TrendingUp size={16}/></button>
          <button onClick={() => setChartType('area')} title="Area" style={{ background: chartType==='area'?'#334155':'transparent', border:'none', color:'#94a3b8', padding:'4px', borderRadius:'4px', cursor:'pointer' }}><Activity size={16}/></button>
        </div>

        <div style={{ width: '1px', height: '20px', background: '#334155', margin: '0 4px' }}></div>

        {/* Indicators */}
        <div style={{ display: 'flex', gap: '4px', position: 'relative', alignItems: 'center' }}>
          <button 
            onClick={() => setShowIndicators(!showIndicators)}
            style={{ background: 'transparent', border: 'none', cursor: 'pointer', padding: '4px', color: '#64748b' }}
            title="Settings"
          >
            <SlidersHorizontal size={14} />
          </button>
          
          {showIndicators && (
            <div style={{ display: 'flex', gap: '4px' }}>
              {indicatorOptions.map(ind => (
                 <button
                    key={ind}
                    onClick={() => toggleIndicator(ind)}
                    style={{
                      background: activeIndicators.includes(ind) ? '#334155' : 'transparent',
                      color: activeIndicators.includes(ind) ? '#60a5fa' : '#64748b',
                      border: '1px solid #334155', padding: '2px 8px', borderRadius: '4px',
                      cursor: 'pointer', fontSize: '11px'
                    }}
                 >
                   {ind}
                 </button>
              ))}
            </div>
          )}
        </div>
        
        <div style={{ flex: 1 }}></div>

        {/* Drawing Tools */}
        <div style={{ display: 'flex', gap: '4px' }}>
          <button 
            onClick={() => setIsDrawMode(!isDrawMode)}
            title="Draw Line"
            style={{ background: isDrawMode?'#3b82f6':'transparent', border:'none', color: isDrawMode?'#fff':'#94a3b8', padding:'4px', borderRadius:'4px', cursor:'pointer' }}
          >
            <PenTool size={16} />
          </button>
          <button onClick={clearDrawings} title="Clear Drawings" style={{ background:'transparent', border:'none', color:'#94a3b8', padding:'4px', borderRadius:'4px', cursor:'pointer' }}>
            <Trash2 size={16} />
          </button>
        </div>
      </div>

      {/* Chart Container */}
      <div style={{ position: 'relative', flex: 1, minHeight: 0 }}>
        <div ref={containerRef} style={{ width: '100%', height: '100%', position: 'absolute', top: 0, left: 0 }} />
        
        {/* Watermark */}
        <div style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          fontSize: '10vw',
          fontWeight: 800,
          color: '#94a3b8',
          opacity: 0.03,
          pointerEvents: 'none',
          zIndex: 0,
          whiteSpace: 'nowrap',
          userSelect: 'none'
        }}>
          {displaySymbol}
        </div>
      </div>
    </div>
  );
}

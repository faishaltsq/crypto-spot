"use client";

import { useEffect, useRef } from "react";
import type { CandlestickData, IChartApi, UTCTimestamp } from "lightweight-charts";

import type { Candle } from "@/types/market";

interface Props {
  candles: Candle[];
}

export function PriceChart({ candles }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let chart: IChartApi | null = null;
    let observer: ResizeObserver | null = null;
    let disposed = false;

    async function render() {
      if (!containerRef.current || candles.length === 0) return;
      const { createChart, CandlestickSeries } = await import("lightweight-charts");
      if (disposed || !containerRef.current) return;

      chart = createChart(containerRef.current, {
        autoSize: true,
        height: 330,
        layout: {
          background: { color: "#111827" },
          textColor: "#cbd5e1",
        },
        grid: {
          vertLines: { color: "#1f2937" },
          horzLines: { color: "#1f2937" },
        },
        rightPriceScale: {
          borderColor: "#334155",
        },
        timeScale: {
          borderColor: "#334155",
          timeVisible: true,
        },
      });

      const series = chart.addSeries(CandlestickSeries, {
        upColor: "#22c55e",
        downColor: "#ef4444",
        borderVisible: false,
        wickUpColor: "#22c55e",
        wickDownColor: "#ef4444",
      });

      const dataMap = new Map<number, CandlestickData>();
      for (const item of candles) {
        if (!item.openTime || !item.open) continue;
        const t = Math.floor(new Date(item.openTime).getTime() / 1000);
        if (!t || Number.isNaN(t)) continue;
        dataMap.set(t, {
          time: t as UTCTimestamp,
          open: item.open,
          high: item.high,
          low: item.low,
          close: item.close,
        });
      }
      const data: CandlestickData[] = Array.from(dataMap.values()).sort(
        (a, b) => (a.time as number) - (b.time as number),
      );

      if (data.length > 0) {
        series.setData(data);
        chart.timeScale().fitContent();
      }

      observer = new ResizeObserver(() => {
        chart?.applyOptions({ width: containerRef.current?.clientWidth ?? 0 });
      });
      observer.observe(containerRef.current);
    }

    void render();
    return () => {
      disposed = true;
      observer?.disconnect();
      chart?.remove();
    };
  }, [candles]);

  if (candles.length === 0) {
    return <div className="emptyChart">Riwayat candle belum cukup.</div>;
  }

  return <div ref={containerRef} className="chart" />;
}

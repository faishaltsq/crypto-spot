"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";

import { getPair, getPerformanceSummary, getQualityStats, getScanner, getSignals } from "@/lib/api";
import { formatCompact, formatPercent, formatPrice, formatTime } from "@/lib/format";
import { useRealtime } from "@/hooks/use-realtime";
import type {
  FeatureSnapshot,
  MarketPairResponse,
  PerformanceSummary,
  QualityStats,
  RealtimeMessage,
  Signal,
} from "@/types/market";
import { PriceChart } from "@/components/price-chart";
import { ScoreBar } from "@/components/score-bar";
import { StatusPill } from "@/components/status-pill";

function notificationSupported(): boolean {
  return typeof window !== "undefined" && "Notification" in window;
}

async function showSignalNotification(signal: Signal, minimumScore: number): Promise<void> {
  if (signal.ruleScore < minimumScore) return;
  if (!notificationSupported() || Notification.permission !== "granted") return;
  const title = `${signal.symbol} ${signal.type}`;
  const body = `Score ${signal.ruleScore.toFixed(1)} | ${signal.primaryTimeframe} | AI review ${signal.ai.decision}`;

  if ("serviceWorker" in navigator) {
    const registration = await navigator.serviceWorker.ready;
    await registration.showNotification(title, {
      body,
      tag: signal.id,
      data: { url: "/" },
    });
    return;
  }
  new Notification(title, { body, tag: signal.id });
}

export function Dashboard() {
  const [scanner, setScanner] = useState<FeatureSnapshot[]>([]);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [selected, setSelected] = useState<string>("BTC_USDT");
  const [tierFilter, setTierFilter] = useState<number>(0);
  const [chartTimeframe, setChartTimeframe] = useState("15m");
  const [pair, setPair] = useState<MarketPairResponse | null>(null);
  const [connected, setConnected] = useState(false);
  const [performance, setPerformance] = useState<PerformanceSummary | null>(null);
  const [qualityStats, setQualityStats] = useState<QualityStats | null>(null);
  const [notificationMinScore, setNotificationMinScore] = useState(80);
  const [loadingPair, setLoadingPair] = useState(false);
  const [error, setError] = useState<string>("");
  const [notifications, setNotifications] = useState<NotificationPermission | "unsupported">("unsupported");

  useEffect(() => {
    if (notificationSupported()) {
      setNotifications(Notification.permission);
    }
  }, []);

  const loadInitial = useCallback(async () => {
    try {
      const [scannerData, signalData, performanceData, statsData] = await Promise.all([
        getScanner(),
        getSignals(50),
        getPerformanceSummary(),
        getQualityStats().catch(() => null),
      ]);
      setScanner(scannerData);
      setSignals(signalData);
      setPerformance(performanceData);
      setQualityStats(statsData);
      setSelected((current) => {
        if (scannerData.length === 0 || scannerData.some((item) => item.symbol === current)) {
          return current;
        }
        return scannerData[0].symbol;
      });
      setError("");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Tidak dapat mengambil data.");
    }
  }, []);

  useEffect(() => {
    void loadInitial();
  }, [loadInitial]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      void getPerformanceSummary().then(setPerformance).catch(() => undefined);
    }, 60_000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      void navigator.serviceWorker.register("/sw.js");
    }
  }, []);

  useEffect(() => {
    let active = true;
    setLoadingPair(true);
    getPair(selected)
      .then((data) => {
        if (active) setPair(data);
      })
      .catch((cause) => {
        if (active) setError(cause instanceof Error ? cause.message : "Pair gagal dimuat.");
      })
      .finally(() => {
        if (active) setLoadingPair(false);
      });
    return () => {
      active = false;
    };
  }, [selected, scanner]);

  const handleRealtime = useCallback((message: RealtimeMessage) => {
    if (message.event === "scanner.snapshot") {
      setScanner(message.data as FeatureSnapshot[]);
    }
    if (message.event === "signal.created") {
      const signal = message.data as Signal;
      setSignals((current) => [signal, ...current].slice(0, 100));
      void showSignalNotification(signal, notificationMinScore);
    }
    if (message.event === "sell.signal.created") {
      const signal = message.data as Signal;
      setSignals((current) => [signal, ...current].slice(0, 100));
      void showSignalNotification(signal, notificationMinScore);
    }
  }, [notificationMinScore]);

  useRealtime({
    onMessage: handleRealtime,
    onStatusChange: setConnected,
  });

  const metrics = useMemo(() => {
    const ready = scanner.filter((item) => item.dataQualityScore >= 60);
    const setups = scanner.filter((item) =>
      ["BUY_SETUP", "BUY_CONFIRMED_CANDIDATE"].includes(item.status),
    );
    const average = ready.length
      ? ready.reduce((sum, item) => sum + item.ruleScore, 0) / ready.length
      : 0;
    return {
      pairs: scanner.length,
      ready: ready.length,
      setups: setups.length,
      average,
    };
  }, [scanner]);

  async function enableNotifications() {
    if (!notificationSupported()) return;
    const permission = await Notification.requestPermission();
    setNotifications(permission);
  }

  const timeframeOrder = ["10s", "1m", "5m", "15m", "30m", "1h", "4h", "8h", "1d", "7d"];
  const availableTimeframes = timeframeOrder.filter(
    (timeframe) => (pair?.market.candles?.[timeframe]?.length ?? 0) > 0,
  );
  const effectiveTimeframe = availableTimeframes.includes(chartTimeframe)
    ? chartTimeframe
    : availableTimeframes[0] ?? chartTimeframe;
  const candles = pair?.market.candles?.[effectiveTimeframe] ?? [];

  const filteredScanner = useMemo(() => {
    if (tierFilter === 0) return scanner;
    return scanner.filter(item => item.tier === tierFilter);
  }, [scanner, tierFilter]);

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">GATE SPOT MARKET MONITOR</p>
          <h1>Crypto Spot Signal</h1>
          <p className="subtitle">
            Paper signal. Tidak memiliki fitur eksekusi order.
          </p>
        </div>
        <div className="topActions">
          <StatusPill tone={connected ? "positive" : "negative"}>
            {connected ? "Realtime connected" : "Realtime disconnected"}
          </StatusPill>
          <label className="scoreControl">
            <span>Min. notifikasi</span>
            <input
              type="number"
              min={60}
              max={100}
              step={1}
              value={notificationMinScore}
              onChange={(event) => setNotificationMinScore(Number(event.target.value) || 80)}
            />
          </label>
          <button
            className="button"
            type="button"
            onClick={enableNotifications}
            disabled={notifications === "unsupported"}
          >
            Notifikasi: {notifications}
          </button>
          <Link
            href="/performance"
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 6,
              padding: "6px 14px",
              borderRadius: 8,
              background: "var(--panel)",
              border: "1px solid var(--border)",
              color: "var(--text)",
              fontSize: 12,
              fontWeight: 600,
              textDecoration: "none",
              transition: "border-color 0.2s",
            }}
          >
            📊 Proof-of-Edge
            {qualityStats && qualityStats.blockedPairs > 0 && (
              <span style={{
                background: "var(--negative)",
                color: "#fff",
                borderRadius: 10,
                padding: "1px 6px",
                fontSize: 10,
                fontWeight: 700,
              }}>
                {qualityStats.blockedPairs} blocked
              </span>
            )}
          </Link>
        </div>
      </header>

      {error && <div className="errorBox">{error}</div>}

      <section className="metricGrid">
        <article className="metricCard">
          <span>Pair dipantau</span>
          <strong>{metrics.pairs}</strong>
        </article>
        <article className="metricCard">
          <span>Data siap</span>
          <strong>{metrics.ready}</strong>
        </article>
        <article className="metricCard">
          <span>Candidate setup</span>
          <strong>{metrics.setups}</strong>
        </article>
        <article className="metricCard">
          <span>Rata-rata score</span>
          <strong>{metrics.average.toFixed(1)}</strong>
        </article>
      </section>

      <section className="performanceStrip">
        <div><span>Total signal</span><strong>{performance?.totalSignals ?? 0}</strong></div>
        <div><span>Sudah dievaluasi</span><strong>{performance?.evaluatedSignals ?? 0}</strong></div>
        <div><span>Target hit rate</span><strong>{formatPercent((performance?.targetHitRate ?? 0) * 100)}</strong></div>
        <div><span>Rata-rata return 1h</span><strong>{formatPercent((performance?.averageReturn1h ?? 0) * 100)}</strong></div>
      </section>

      <section className="panel">
        <div className="panelHeader">
          <div>
            <h2>Realtime scanner</h2>
            <p>Score menggabungkan trend, volume, order flow, liquidity, dan kualitas data.</p>
          </div>
          <div className="tierFilters">
            <button className={`button ${tierFilter === 0 ? "active" : "outline"}`} onClick={() => setTierFilter(0)}>All</button>
            <button className={`button ${tierFilter === 1 ? "active" : "outline"}`} onClick={() => setTierFilter(1)}>Tier A (Top 30)</button>
            <button className={`button ${tierFilter === 2 ? "active" : "outline"}`} onClick={() => setTierFilter(2)}>Tier B (31-80)</button>
            <button className={`button ${tierFilter === 3 ? "active" : "outline"}`} onClick={() => setTierFilter(3)}>Tier C (81-150)</button>
          </div>
        </div>
        <div className="tableWrap">
          <table>
            <thead>
              <tr>
                <th>Pair</th>
                <th>Harga</th>
                <th>24h</th>
                <th>RVOL 1m</th>
                <th>Delta 1m</th>
                <th>Book imbalance</th>
                <th>Spread</th>
                <th>Spoof</th>
                <th>Score</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {filteredScanner.map((item) => (
                <tr
                  key={item.symbol}
                  className={selected === item.symbol ? "selectedRow" : ""}
                  onClick={() => setSelected(item.symbol)}
                >
                  <td>
                    <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                      <strong>{item.symbol}</strong>
                      {item.dataSource === "MOCK" && (
                        <span style={{
                          fontSize: 9,
                          padding: "2px 4px",
                          borderRadius: 4,
                          background: "var(--warning)",
                          color: "#fff",
                          fontWeight: 700
                        }}>
                          MOCK
                        </span>
                      )}
                    </div>
                  </td>
                  <td>{formatPrice(item.price)}</td>
                  <td className={item.change24hPercent >= 0 ? "positiveText" : "negativeText"}>
                    {formatPercent(item.change24hPercent)}
                  </td>
                  <td>{item.relativeVolume1m.toFixed(2)}x</td>
                  <td>{formatPercent(item.volumeDeltaRatio1m * 100)}</td>
                  <td>{item.orderbookImbalance.toFixed(3)}</td>
                  <td>{item.spreadBps.toFixed(2)} bps</td>
                  <td>{item.spoofScore.toFixed(1)}</td>
                  <td><ScoreBar value={item.ruleScore} /></td>
                  <td>
                    <StatusPill
                      tone={
                        item.status.includes("BUY")
                          ? "positive"
                          : item.status.includes("DATA") || item.status.includes("BLOCKED")
                            ? "warning"
                            : "neutral"
                      }
                      title={item.missingFeatures?.join(", ") || item.blockedReasons?.join(", ")}
                    >
                      {item.status}
                    </StatusPill>
                  </td>
                </tr>
              ))}
              {scanner.length === 0 && (
                <tr>
                  <td colSpan={10} className="emptyCell">
                    Menunggu data Gate dan histori candle.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="detailGrid">
        <article className="panel">
          <div className="panelHeader">
            <div>
              <h2>{selected} chart</h2>
              <p>Candlestick SPOT dari timeframe yang dipilih.</p>
            </div>
            <div className="chartControls">
              <label>
                <span>Timeframe</span>
                <select
                  value={effectiveTimeframe}
                  onChange={(event) => setChartTimeframe(event.target.value)}
                >
                  {availableTimeframes.map((timeframe) => (
                    <option key={timeframe} value={timeframe}>{timeframe}</option>
                  ))}
                </select>
              </label>
              {loadingPair && <span className="muted">Memuat...</span>}
            </div>
          </div>
          <PriceChart candles={candles} />
        </article>

        <article className="panel">
          <div className="panelHeader">
            <div>
              <h2>Pair diagnostics</h2>
              <p>Kondisi order book dan alasan score saat ini.</p>
            </div>
          </div>
          {pair?.feature ? (
            <div className="diagnostics">
              <div className="diagnosticRow"><span>Harga</span><strong>{formatPrice(pair.feature.price)}</strong></div>
              <div className="diagnosticRow"><span>Volume 24h</span><strong>{formatCompact(pair.feature.quoteVolume24h)}</strong></div>
              <div className="diagnosticRow"><span>Bid depth</span><strong>{formatCompact(pair.feature.bidDepthQuote)}</strong></div>
              <div className="diagnosticRow"><span>Ask depth</span><strong>{formatCompact(pair.feature.askDepthQuote)}</strong></div>
              <div className="diagnosticRow"><span>Data quality</span><strong>{pair.feature.dataQualityScore.toFixed(1)}</strong></div>
              <div className="diagnosticRow"><span>Liquidity</span><strong>{pair.feature.liquidityScore.toFixed(1)}</strong></div>
              <div className="tagGroup">
                {(pair.feature.reasons || []).map((reason) => (
                  <StatusPill key={reason} tone="positive">{reason}</StatusPill>
                ))}
                {(pair.feature.riskFlags || []).map((flag) => (
                  <StatusPill key={flag} tone="warning">{flag}</StatusPill>
                ))}
              </div>
              <div className="trendGrid">
                {Object.entries(pair.feature.trendByTimeframe || {}).map(([tf, trend]) => (
                  <div key={tf} className="trendItem">
                    <span>{tf}</span>
                    <strong className={`${trend}Text`}>{trend}</strong>
                  </div>
                ))}
              </div>
              <div className="orderbookGrid">
                <div>
                  <h3>Top bids</h3>
                  {(pair.market?.topBids || []).slice(0, 8).map((level) => (
                    <div className="bookRow" key={`bid-${level.price}`}>
                      <span className="positiveText">{formatPrice(level.price)}</span>
                      <strong>{formatCompact(level.amount)}</strong>
                    </div>
                  ))}
                </div>
                <div>
                  <h3>Top asks</h3>
                  {(pair.market?.topAsks || []).slice(0, 8).map((level) => (
                    <div className="bookRow" key={`ask-${level.price}`}>
                      <span className="negativeText">{formatPrice(level.price)}</span>
                      <strong>{formatCompact(level.amount)}</strong>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <p className="muted">Data pair belum tersedia.</p>
          )}
        </article>
      </section>

      <section className="panel">
        <div className="panelHeader">
          <div>
            <h2>Signal history</h2>
            <p>Candidate yang lolos threshold, cooldown, dan review layer.</p>
          </div>
        </div>
        <div className="signalList">
          {(signals || []).map((signal) => (
            <article className="signalCard" key={signal.id}>
              <div className="signalMain">
                <div>
                  <strong>{signal.symbol}</strong>
                  <span>{signal.type} · {signal.primaryTimeframe}</span>
                </div>
                <ScoreBar value={signal.ruleScore} />
              </div>
              <div className="signalPrices">
                <span>Entry <strong>{formatPrice(signal.entryPrice)}</strong></span>
                <span>Invalidation <strong>{formatPrice(signal.invalidationPrice)}</strong></span>
                <span>Target 1 <strong>{formatPrice(signal.targetPrice1)}</strong></span>
                <span>Target 2 <strong>{formatPrice(signal.targetPrice2)}</strong></span>
              </div>
              <p>{signal.ai?.summary || "Tidak ada summary AI."}</p>
              <small>{formatTime(signal.createdAt)} · {signal.ai?.provider || "N/A"}/{signal.ai?.model || "N/A"}</small>
            </article>
          ))}
          {signals.length === 0 && (
            <p className="muted">Belum ada signal yang disimpan.</p>
          )}
        </div>
      </section>
    </main>
  );
}

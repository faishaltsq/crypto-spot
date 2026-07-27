'use client';

import { useEffect, useRef } from 'react';
import { allowsWatchlistNotification, loadWatchlistState } from '@/lib/watchlist';
import { useMarketStore } from '@/stores/market';

export function WatchlistNotifications() {
  const signals = useMarketStore((state) => state.signals);
  const notifiedIds = useRef(new Set<string>());
  const initialized = useRef(false);

  useEffect(() => {
    if (!initialized.current) {
      for (const signal of signals) notifiedIds.current.add(signal.id);
      initialized.current = true;
      return;
    }
    for (const signal of signals) {
      if (notifiedIds.current.has(signal.id)) continue;
      notifiedIds.current.add(signal.id);
      const state = loadWatchlistState();
      const matches = state?.watchlists.some((watchlist) =>
        watchlist.pairs.some((pair) => allowsWatchlistNotification(pair, signal)),
      );
      if (!matches || typeof Notification === 'undefined' || Notification.permission !== 'granted') continue;

      const title = `${signal.symbol} ${signal.type}`;
      const body = `Score ${signal.ruleScore.toFixed(1)} | ${signal.primaryTimeframe}`;
      if ('serviceWorker' in navigator) {
        void navigator.serviceWorker.ready.then((registration) =>
          registration.showNotification(title, { body, tag: `watchlist-${signal.id}`, data: { url: '/watchlist' } }),
        );
      } else {
        new Notification(title, { body, tag: `watchlist-${signal.id}` });
      }
    }
  }, [signals]);

  return null;
}

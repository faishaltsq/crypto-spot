'use client';

import { useEffect, useRef } from 'react';
import { useRealtime } from '@/hooks/use-realtime';
import { useMarketStore } from '@/stores/market';

export function GlobalRealtime() {
  const updatePair = useMarketStore(state => state.updatePair);
  const updateSignal = useMarketStore(state => state.updateSignal);
  const updateSellSignal = useMarketStore(state => state.updateSellSignal);

  const socketRef = useRef<WebSocket | null>(null);
  const comparePairsRef = useRef(new Set<string>());

  useEffect(() => {
    const handleSubscription = (event: Event) => {
      const detail = (event as CustomEvent<{ action: 'subscribe' | 'unsubscribe'; pairs: string[] }>).detail;
      if (!detail) return;
      for (const pair of detail.pairs) {
        if (detail.action === 'subscribe') comparePairsRef.current.add(pair);
        else comparePairsRef.current.delete(pair);
      }
      if (socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(JSON.stringify({ channel: 'compare', ...detail }));
    };
    window.addEventListener('compare-subscription', handleSubscription);
    return () => window.removeEventListener('compare-subscription', handleSubscription);
  }, []);

  useRealtime({
    onMessage: (message) => {
      switch (message.event) {
        case 'scanner.snapshot':
          updatePair(message.data as any);
          break;
        case 'compare.snapshot':
          window.dispatchEvent(new CustomEvent('compare-update', { detail: message.data }));
          break;
        case 'quality.snapshot':
          window.dispatchEvent(new CustomEvent('quality-updated'));
          break;
        case 'signal.created':
          // Play sound on new signal
          const audio = new Audio('/notification.mp3');
          audio.play().catch(e => console.error("Audio playback failed", e));
          updateSignal(message.data as any);
          break;
        case 'sell.signal.created':
          // Play sound on new SELL signal too
          const audioSell = new Audio('/notification.mp3');
          audioSell.play().catch(e => console.error("Audio playback failed", e));
          updateSellSignal(message.data as any);
          break;
        case 'signal.update':
      }
    },
    onStatusChange: (connected) => {
      if (!connected) socketRef.current = null;
    },
    onSocketChange: (socket) => {
      socketRef.current = socket;
      if (socket && comparePairsRef.current.size > 0) socket.send(JSON.stringify({ channel: 'compare', action: 'subscribe', pairs: [...comparePairsRef.current] }));
    }
  });

  return null;
}

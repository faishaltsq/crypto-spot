'use client';

import { useEffect, useRef } from 'react';
import { useRealtime } from '@/hooks/use-realtime';
import { useMarketStore } from '@/stores/market';

// isConfirmed decides whether an incoming signal payload should trigger the
// audible notification. We chime only for CONFIRMED-status signals — the ones
// shown as actionable in the panel — matching the backend lifecycle contract
// (status "CONFIRMED" / lifecycleGroup "CONFIRMED"). BUY_CONFIRMED and
// SELL_CONFIRMED both land on status "CONFIRMED".
function isConfirmed(payload: any): boolean {
  if (!payload) return false;
  const status = String(payload.status ?? '').toUpperCase().trim();
  const lifecycle = String(payload.lifecycleGroup ?? '').toUpperCase().trim();
  return status === 'CONFIRMED' || lifecycle === 'CONFIRMED';
}

export function GlobalRealtime() {
  const updatePair = useMarketStore(state => state.updatePair);
  const updateSignal = useMarketStore(state => state.updateSignal);
  const updateSellSignal = useMarketStore(state => state.updateSellSignal);

  const socketRef = useRef<WebSocket | null>(null);
  const comparePairsRef = useRef(new Set<string>());

  // A single reusable Audio element, "unlocked" on the first user gesture.
  // Browsers block audio.play() until the user has interacted with the page
  // (the "play() failed because the user didn't interact" error). We prime
  // the element on the first pointer/key/touch event so later signal-driven
  // playback is allowed. If no interaction has happened yet, we skip playback
  // silently instead of throwing.
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const audioUnlockedRef = useRef(false);

  useEffect(() => {
    audioRef.current = new Audio('/notification.mp3');
    const unlock = () => {
      const el = audioRef.current;
      if (!el) return;
      // Play then immediately pause+reset to satisfy the gesture requirement.
      el.muted = true;
      el.play().then(() => {
        el.pause();
        el.currentTime = 0;
        el.muted = false;
        audioUnlockedRef.current = true;
      }).catch(() => {
        // Still locked; will retry on next gesture.
      });
    };
    const events: (keyof WindowEventMap)[] = ['pointerdown', 'keydown', 'touchstart'];
    events.forEach(evt => window.addEventListener(evt, unlock, { once: false }));
    return () => events.forEach(evt => window.removeEventListener(evt, unlock));
  }, []);

  const playNotificationSound = () => {
    if (!audioUnlockedRef.current || !audioRef.current) return;
    audioRef.current.currentTime = 0;
    audioRef.current.play().catch(() => {
      // Autoplay still blocked (e.g. gesture expired); ignore.
    });
  };

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
        case 'signal.created': {
          const payload = message.data as any;
          // Only chime for CONFIRMED signals (the ones surfaced as actionable
          // in the panel). SETUP/BLOCKED/etc. update the store silently.
          if (isConfirmed(payload)) playNotificationSound();
          updateSignal(payload);
          break;
        }
        case 'sell.signal.created': {
          const payload = message.data as any;
          if (isConfirmed(payload)) playNotificationSound();
          updateSellSignal(payload);
          break;
        }
        case 'signal.updated': {
          // Backend-driven lifecycle transition (target hit / invalidated /
          // expired / closed) for BUY or SELL family signals. Route by
          // shape: SELL-family payloads carry sellScore, BUY payloads
          // don't.
          const payload = message.data as any;
          if (payload && typeof payload.sellScore === 'number') {
            updateSellSignal(payload);
          } else {
            updateSignal(payload);
          }
          break;
        }
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

'use client';

import { useEffect } from 'react';
import { useRealtime } from '@/hooks/use-realtime';
import { useMarketStore } from '@/stores/market';

export function GlobalRealtime() {
  const updatePair = useMarketStore(state => state.updatePair);
  const updateSignal = useMarketStore(state => state.updateSignal);

  useRealtime({
    onMessage: (message) => {
      switch (message.event) {
        case 'scanner.snapshot':
          updatePair(message.data as any);
          break;
        case 'signal.new':
          // Play sound on new signal
          const audio = new Audio('/notification.mp3');
          audio.play().catch(e => console.error("Audio playback failed", e));
          updateSignal(message.data as any);
          break;
        case 'signal.update':
      }
    },
    onStatusChange: (connected) => {
      // Could dispatch to a connection status store
    }
  });

  return null;
}

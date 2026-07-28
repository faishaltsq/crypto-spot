import { create } from 'zustand';
import { FeatureSnapshot, Signal, SellSignalDetail } from '@/types/market';
import { getScanner, getSignals, sellApi } from '@/lib/api';
import { normalizeSignal, normalizeSignals } from '@/lib/signal-normalize';
import { isSignalActive } from '@/lib/signal-status';

interface MarketState {
  scanner: Record<string, FeatureSnapshot>;
  signals: Signal[];
  sellSignals: SellSignalDetail[];
  scannerArray: FeatureSnapshot[];
  isLoading: boolean;
  error: string | null;
  initializeScanner: () => Promise<void>;
  initializeSignals: () => Promise<void>;
  updatePair: (pair: FeatureSnapshot) => void;
  updateSignal: (signal: Signal) => void;
  updateSellSignal: (signal: SellSignalDetail) => void;
}

let pendingPairs: Record<string, FeatureSnapshot> = {};
let flushTimeout: ReturnType<typeof setTimeout> | null = null;
const FLUSH_INTERVAL = 500;

export const useMarketStore = create<MarketState>((set, get) => ({
  scanner: {},
  scannerArray: [],
  signals: [],
  sellSignals: [],
  isLoading: false,
  error: null,

  initializeScanner: async () => {
    set({ isLoading: true, error: null });
    try {
      const scannerData = await getScanner();
      const scannerRecord: Record<string, FeatureSnapshot> = {};
      
      for (const item of scannerData) {
        scannerRecord[item.symbol] = item;
      }
      
      set({ 
        scanner: scannerRecord, 
        scannerArray: scannerData,
        isLoading: false 
      });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to fetch scanner data',
        isLoading: false
      });
    }
  },

  initializeSignals: async () => {
    try {
      const [globalSignals, globalSellSignals] = await Promise.all([
        getSignals(100),
        sellApi.listSignals(undefined, 100).then(res => res.signals)
      ]);
      set({
        signals: normalizeSignals(globalSignals),
        sellSignals: normalizeSignals(globalSellSignals) as SellSignalDetail[],
      });
    } catch (error) {
      console.error('Failed to fetch global signals:', error);
    }
  },

  updatePair: (pair: FeatureSnapshot) => {
    pendingPairs[pair.symbol] = pair;

    if (!flushTimeout) {
      flushTimeout = setTimeout(() => {
        set((state) => {
          const pairsToApply = pendingPairs;
          pendingPairs = {};
          flushTimeout = null;

          const newScanner = { ...state.scanner, ...pairsToApply };

          // NOTE: signal lifecycle transitions (target hit / invalidation /
          // expiry) are decided exclusively by the backend (see
          // backend/internal/domain/signal_status.go + outcomeLoop /
          // sell.OutcomeEvaluator in backend/cmd/server/main.go), which
          // broadcasts a `signal.updated` event over the same WebSocket
          // whenever a signal's status actually changes. The frontend must
          // never guess a status transition from a price tick — doing so
          // risks disagreeing with the backend's persisted status and
          // showing a signal as closed/invalidated before the backend
          // agrees, or vice versa.

          return {
            scanner: newScanner,
            scannerArray: Object.values(newScanner),
          };
        });
      }, FLUSH_INTERVAL);
    }
  },

  updateSignal: (signal: Signal) => {
    const normalized = normalizeSignal(signal);
    set((state) => {
      const existingIdx = state.signals.findIndex(s => s.id === normalized.id);
      if (existingIdx >= 0) {
        const newSignals = [...state.signals];
        newSignals[existingIdx] = normalized;
        return { signals: newSignals };
      }
      return { signals: [normalized, ...state.signals].slice(0, 100) };
    });
  },

  updateSellSignal: (signal: SellSignalDetail) => {
    const normalized = normalizeSignal(signal) as SellSignalDetail;
    set((state) => {
      const existingIdx = state.sellSignals.findIndex(s => s.id === normalized.id);
      if (existingIdx >= 0) {
        const newSignals = [...state.sellSignals];
        newSignals[existingIdx] = normalized;
        return { sellSignals: newSignals };
      }
      return { sellSignals: [normalized, ...state.sellSignals].slice(0, 100) };
    });
  }
}));

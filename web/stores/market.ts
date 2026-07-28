import { create } from 'zustand';
import { FeatureSnapshot, Signal, SellSignalDetail } from '@/types/market';
import { getScanner, getSignals, sellApi } from '@/lib/api';

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
      set({ signals: globalSignals, sellSignals: globalSellSignals });
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
          
          // Auto-evaluate active signals based on new price
          let signalsUpdated = false;
          const newSignals = state.signals.map(s => {
            if (s.status === 'CLOSED' || s.status === 'INVALIDATED' || !pairsToApply[s.symbol]) {
              return s;
            }
            
            const currentPair = pairsToApply[s.symbol];
            const isLong = s.type.includes('BUY') || s.type.includes('LONG');
            let newStatus = s.status;
            
            if (isLong) {
              if (currentPair.price >= s.targetPrice1) newStatus = 'CLOSED';
              else if (currentPair.price <= s.invalidationPrice) newStatus = 'INVALIDATED';
            } else {
              if (currentPair.price <= s.targetPrice1) newStatus = 'CLOSED';
              else if (currentPair.price >= s.invalidationPrice) newStatus = 'INVALIDATED';
            }
            
            if (newStatus !== s.status) {
              signalsUpdated = true;
              return { ...s, status: newStatus };
            }
            return s;
          });

          return {
            scanner: newScanner,
            scannerArray: Object.values(newScanner),
            ...(signalsUpdated ? { signals: newSignals } : {})
          };
        });
      }, FLUSH_INTERVAL);
    }
  },

  updateSignal: (signal: Signal) => {
    set((state) => {
      const existingIdx = state.signals.findIndex(s => s.id === signal.id);
      if (existingIdx >= 0) {
        const newSignals = [...state.signals];
        newSignals[existingIdx] = signal;
        return { signals: newSignals };
      }
      return { signals: [signal, ...state.signals].slice(0, 100) };
    });
  },

  updateSellSignal: (signal: SellSignalDetail) => {
    set((state) => {
      const existingIdx = state.sellSignals.findIndex(s => s.id === signal.id);
      if (existingIdx >= 0) {
        const newSignals = [...state.sellSignals];
        newSignals[existingIdx] = signal;
        return { sellSignals: newSignals };
      }
      return { sellSignals: [signal, ...state.sellSignals].slice(0, 100) };
    });
  }
}));

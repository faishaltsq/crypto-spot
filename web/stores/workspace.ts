import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface WorkspaceState {
  showLeftPanel: boolean;
  showRightPanel: boolean;
  showDiagnostic: boolean;
  showVolume: boolean;
  showOrderBook: boolean;
  theme: 'dark' | 'light' | 'system';
  density: 'compact' | 'comfortable';
  activeChartTimeframe: string;
  activeIndicators: string[];
  watchlist: string[];
  toggleLeftPanel: () => void;
  toggleRightPanel: () => void;
  toggleDiagnostic: () => void;
  setTheme: (theme: 'dark' | 'light' | 'system') => void;
  setDensity: (density: 'compact' | 'comfortable') => void;
  setActiveChartTimeframe: (tf: string) => void;
  toggleIndicator: (indicator: string) => void;
  toggleWatchlist: (symbol: string) => void;
  resetLayout: () => void;
}

const initialState = {
  showLeftPanel: true,
  showRightPanel: true,
  showDiagnostic: true,
  showVolume: true,
  showOrderBook: false,
  theme: 'dark' as const,
  density: 'comfortable' as const,
  activeChartTimeframe: '15m',
  activeIndicators: ['Volume', 'EMA 20'],
  watchlist: [],
};

export const useWorkspace = create<WorkspaceState>()(
  persist(
    (set) => ({
      ...initialState,
      toggleLeftPanel: () => set((state) => ({ showLeftPanel: !state.showLeftPanel })),
      toggleRightPanel: () => set((state) => ({ showRightPanel: !state.showRightPanel })),
      toggleDiagnostic: () => set((state) => ({ showDiagnostic: !state.showDiagnostic })),
      setTheme: (theme) => set({ theme }),
      setDensity: (density) => set({ density }),
      setActiveChartTimeframe: (activeChartTimeframe) => set({ activeChartTimeframe }),
      toggleIndicator: (indicator) => set((state) => {
        const has = state.activeIndicators.includes(indicator);
        return {
          activeIndicators: has
            ? state.activeIndicators.filter((i) => i !== indicator)
            : [...state.activeIndicators, indicator],
        };
      }),
      toggleWatchlist: (symbol) => set((state) => {
        const has = state.watchlist.includes(symbol);
        return {
          watchlist: has
            ? state.watchlist.filter((s) => s !== symbol)
            : [...state.watchlist, symbol],
        };
      }),
      resetLayout: () => set(initialState),
    }),
    {
      name: 'crypto-spot-signal-workspace',
    }
  )
);

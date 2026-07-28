export type UserPreferences = Record<string, unknown>;

export const defaultPreferences: UserPreferences = {
  theme: 'dark', density: 'comfortable', fontSize: 'medium', reduceMotion: false, numberFormat: 'locale', timezone: 'system', currencyDisplay: 'USDT', positiveColor: 'green', negativeColor: 'red', chartWatermark: true, decimalPrecision: 'auto',
  showLeftPairPanel: true, showRightSignalPanel: true, showDiagnostic: true, showOrderBook: false, showRecentTrades: true, defaultPanelSizes: 'balanced', defaultPair: 'BTC_USDT', defaultTimeframe: '15m', defaultTerminalRoute: '/terminal/BTC_USDT',
  chartProvider: 'lightweight', chartType: 'candles', defaultIndicators: ['Volume', 'EMA 20'], signalOverlay: true, drawingPersistence: true, autoScale: true, logarithmicScale: false,
  browserNotifications: false, sound: false, minimumSignalScore: 80, allowedSignalStates: ['CONFIRMED'], allowedRiskLevels: ['LOW', 'MEDIUM'], allowedTimeframes: ['15m'], maximumNotificationsPerHour: 20, digestMode: 'off', quietHours: false, systemAlerts: true, aiAlerts: false, staleDataAlerts: true, orderBookResyncAlerts: true,
  defaultWatchlist: 'Default', watchlistTimeframe: '15m', defaultAlertScore: 80, defaultRiskFilter: 'LOW,MEDIUM',
};

const key = 'crypto-spot-signal-preferences';
export const localPreferenceRepository = {
  load(): UserPreferences { if (typeof window === 'undefined') return defaultPreferences; try { const raw = window.localStorage.getItem(key); return { ...defaultPreferences, ...(raw ? JSON.parse(raw) : {}) }; } catch { return { ...defaultPreferences }; } },
  save(preferences: UserPreferences): void { window.localStorage.setItem(key, JSON.stringify(preferences)); },
  reset(): UserPreferences { window.localStorage.removeItem(key); return { ...defaultPreferences }; },
};

export type SystemSettings = Record<string, unknown>;
export type AdminSetting = { key: string; value: unknown; reloadMode: string; restartRequired: boolean };
const api = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
async function request<T>(path: string, init?: RequestInit): Promise<T> { const response = await fetch(`${api}${path}`, { cache: 'no-store', ...init, headers: { 'Content-Type': 'application/json', 'X-User-Role': 'admin', 'X-User-ID': '00000000-0000-0000-0000-000000000000', ...init?.headers } }); if (!response.ok) throw new Error(`Request failed: ${response.status}`); return response.json() as Promise<T>; }
export const settingsApi = {
  preferences: () => request<{ preferences: UserPreferences; storage: string }>('/api/v1/settings/preferences'),
  savePreferences: (preferences: UserPreferences) => request<{ preferences: UserPreferences; storage: string }>('/api/v1/settings/preferences', { method: 'PUT', body: JSON.stringify({ preferences }) }),
  system: () => request<SystemSettings>('/api/v1/settings/system'),
  admin: () => request<{ settings: AdminSetting[]; allowedKeys: string[] }>('/api/v1/admin/settings'),
  history: () => request<{ versions: Array<Record<string, unknown>>; auditLogs: Array<Record<string, unknown>> }>('/api/v1/admin/settings/history'),
  saveAdmin: (settings: Record<string, unknown>, reason: string) => request<{ settings: AdminSetting[]; validation: string }>('/api/v1/admin/settings', { method: 'PUT', body: JSON.stringify({ settings, reason }) }),
  resetAdmin: (reason: string) => request<{ result: string }>('/api/v1/admin/settings/reset', { method: 'POST', body: JSON.stringify({ reason }) }),
};

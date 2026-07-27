import type { Signal } from "@/types/market";

export type RiskLevel = "LOW" | "MEDIUM" | "HIGH";
export type WatchlistDensity = "compact" | "comfortable";

export interface WatchlistPair {
  id: string;
  symbol: string;
  position: number;
  isFavorite: boolean;
  isPinned: boolean;
  isMuted: boolean;
  preferredTimeframe: string;
  minimumSignalScore: number;
  riskLevels: RiskLevel[];
  signalTypes: string[];
  notificationEnabled: boolean;
  quietHoursStart: string;
  quietHoursEnd: string;
  tags: string[];
  note: string;
  createdAt: string;
  updatedAt: string;
}

export interface Watchlist {
  id: string;
  name: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
  pairs: WatchlistPair[];
}

export interface WatchlistState {
  watchlists: Watchlist[];
  selectedWatchlistId: string;
  density: WatchlistDensity;
}

export interface WatchlistRepository {
  load(): WatchlistState;
  save(state: WatchlistState): void;
  mode: "local" | "backend";
}

export const WATCHLIST_STORAGE_KEY = "crypto-spot-signal-watchlists-v1";

const defaultPair = (symbol: string, position: number): WatchlistPair => {
  const now = new Date().toISOString();
  return {
    id: cryptoId(),
    symbol,
    position,
    isFavorite: true,
    isPinned: false,
    isMuted: false,
    preferredTimeframe: "15m",
    minimumSignalScore: 85,
    riskLevels: ["LOW", "MEDIUM"],
    signalTypes: ["BUY_CONFIRMED"],
    notificationEnabled: false,
    quietHoursStart: "23:00",
    quietHoursEnd: "07:00",
    tags: [],
    note: "",
    createdAt: now,
    updatedAt: now,
  };
};

export function createWatchlist(name: string, isDefault = false): Watchlist {
  const now = new Date().toISOString();
  return { id: cryptoId(), name: name.trim() || "Untitled watchlist", isDefault, createdAt: now, updatedAt: now, pairs: [] };
}

export function initialWatchlistState(): WatchlistState {
  const watchlist = createWatchlist("My watchlist", true);
  watchlist.pairs = ["BTC_USDT", "ETH_USDT", "SOL_USDT"].map(defaultPair);
  return { watchlists: [watchlist], selectedWatchlistId: watchlist.id, density: "comfortable" };
}

export const localWatchlistRepository: WatchlistRepository = {
  mode: "local",
  load() {
    if (typeof window === "undefined") return initialWatchlistState();
    const raw = window.localStorage.getItem(WATCHLIST_STORAGE_KEY);
    if (!raw) return initialWatchlistState();
    const parsed = JSON.parse(raw) as WatchlistState;
    if (!Array.isArray(parsed.watchlists) || parsed.watchlists.length === 0) return initialWatchlistState();
    return {
      watchlists: parsed.watchlists.map(normalizeWatchlist),
      selectedWatchlistId: parsed.selectedWatchlistId,
      density: parsed.density === "compact" ? "compact" : "comfortable",
    };
  },
  save(state) {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(WATCHLIST_STORAGE_KEY, JSON.stringify(state));
  },
};

function normalizeWatchlist(watchlist: Watchlist): Watchlist {
  return {
    ...watchlist,
    pairs: (watchlist.pairs ?? []).map((pair, position) => ({
      ...defaultPair(pair.symbol, position),
      ...pair,
      position,
      tags: pair.tags ?? [],
      riskLevels: pair.riskLevels ?? ["LOW", "MEDIUM"],
      signalTypes: pair.signalTypes ?? ["BUY_CONFIRMED"],
    })),
  };
}

function cryptoId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function addPair(watchlist: Watchlist, symbol: string): Watchlist {
  const normalized = symbol.trim().toUpperCase().replace("/", "_");
  if (!normalized) throw new Error("Pair wajib diisi.");
  if (watchlist.pairs.some((pair) => pair.symbol === normalized)) throw new Error("Pair sudah ada di watchlist ini.");
  return { ...watchlist, updatedAt: new Date().toISOString(), pairs: [...watchlist.pairs, defaultPair(normalized, watchlist.pairs.length)] };
}

export function reorderPairs(pairs: WatchlistPair[], sourceIndex: number, destinationIndex: number): WatchlistPair[] {
  if (sourceIndex === destinationIndex || sourceIndex < 0 || destinationIndex < 0 || sourceIndex >= pairs.length || destinationIndex >= pairs.length) return pairs;
  const next = [...pairs];
  const [moved] = next.splice(sourceIndex, 1);
  next.splice(destinationIndex, 0, moved);
  return next.map((pair, position) => ({ ...pair, position, updatedAt: new Date().toISOString() }));
}

export function isWithinQuietHours(now: Date, start: string, end: string): boolean {
  if (!/^\d{2}:\d{2}$/.test(start) || !/^\d{2}:\d{2}$/.test(end) || start === end) return false;
  const minutes = now.getHours() * 60 + now.getMinutes();
  const toMinutes = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3, 5));
  const startMinutes = toMinutes(start);
  const endMinutes = toMinutes(end);
  return startMinutes < endMinutes
    ? minutes >= startMinutes && minutes < endMinutes
    : minutes >= startMinutes || minutes < endMinutes;
}

export function allowsWatchlistNotification(pair: WatchlistPair, signal: Signal, now = new Date()): boolean {
  if (!pair.notificationEnabled || pair.isMuted || signal.symbol !== pair.symbol) return false;
  if (signal.ruleScore < pair.minimumSignalScore) return false;
  if (pair.preferredTimeframe && signal.primaryTimeframe !== pair.preferredTimeframe) return false;
  if (pair.signalTypes.length > 0 && !pair.signalTypes.includes(signal.type)) return false;
  const risk = signal.riskFlags.includes("HIGH") ? "HIGH" : signal.riskFlags.includes("MEDIUM") ? "MEDIUM" : "LOW";
  if (pair.riskLevels.length > 0 && !pair.riskLevels.includes(risk)) return false;
  return !isWithinQuietHours(now, pair.quietHoursStart, pair.quietHoursEnd);
}

export function loadWatchlistState(): WatchlistState | null {
  try {
    return localWatchlistRepository.load();
  } catch {
    return null;
  }
}

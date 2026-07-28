import type { Signal } from "@/types/market";

const TERMINAL_STATUSES = new Set(["CLOSED", "INVALIDATED", "EXPIRED", "BLOCKED", "SUPPRESSED"]);

function fallbackIsActive(status: string, expiresAt?: string): boolean {
  if (TERMINAL_STATUSES.has((status ?? "").toUpperCase().trim())) return false;
  if (expiresAt && Date.now() > new Date(expiresAt).getTime()) return false;
  return true;
}

function fallbackDirection(type: string): "BUY" | "SELL" {
  const t = (type ?? "").toUpperCase();
  if (t.startsWith("BUY")) return "BUY";
  return "SELL";
}

function fallbackStrategy(type: string): Signal["strategy"] {
  switch ((type ?? "").toUpperCase()) {
    case "BUY_SETUP":
    case "BUY_CONFIRMED":
      return "ENTRY_BUY";
    case "TAKE_PROFIT_SUGGESTED":
      return "TAKE_PROFIT";
    case "EXIT_WARNING":
      return "EXIT_WARNING";
    case "AVOID_ENTRY":
      return "AVOID_ENTRY";
    default:
      return "PROTECTIVE_SELL";
  }
}

function fallbackLifecycleGroup(status: string): Signal["lifecycleGroup"] {
  const s = (status ?? "").toUpperCase().trim();
  const known: Signal["lifecycleGroup"][] = ["SETUP", "CONFIRMED", "ACTIVE", "BLOCKED", "INVALIDATED", "EXPIRED", "CLOSED"];
  if ((known as string[]).includes(s)) return s as Signal["lifecycleGroup"];
  if (s === "SUPPRESSED") return "BLOCKED";
  return "WATCH";
}

/**
 * Defensive normalizer for any raw signal payload coming from REST or
 * WebSocket. The backend (see backend/internal/domain/signal_status.go)
 * always populates isActive/direction/strategy/lifecycleGroup via
 * Signal.Enrich() before serializing — this function's fallback branch
 * should never actually run against a live backend. It exists only to
 * protect against stale cached payloads (e.g. a signal object persisted
 * to localStorage before this contract existed) so the UI degrades
 * gracefully instead of crashing on `undefined.isActive`.
 *
 * Any fallback usage is logged in development so contract violations are
 * caught early rather than silently masked.
 */
export function normalizeSignal<T extends Signal>(raw: T): T {
  if (
    typeof raw.isActive === "boolean" &&
    (raw.direction === "BUY" || raw.direction === "SELL") &&
    typeof raw.strategy === "string" &&
    typeof raw.lifecycleGroup === "string"
  ) {
    return raw;
  }

  if (process.env.NODE_ENV !== "production") {
    // eslint-disable-next-line no-console
    console.warn(
      `[signal-normalize] signal ${raw.id ?? "(no id)"} is missing the backend lifecycle contract fields; falling back to client-side derivation. This indicates a stale payload or a backend contract regression.`,
    );
  }

  return {
    ...raw,
    isActive: raw.isActive ?? fallbackIsActive(raw.status, raw.expiresAt),
    direction: raw.direction ?? fallbackDirection(raw.type),
    strategy: raw.strategy ?? fallbackStrategy(raw.type),
    lifecycleGroup: raw.lifecycleGroup ?? fallbackLifecycleGroup(raw.status),
  };
}

export function normalizeSignals<T extends Signal>(raws: T[] | null | undefined): T[] {
  return (raws ?? []).map(normalizeSignal);
}

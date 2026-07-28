import type { Signal } from "@/types/market";

/**
 * Single source of truth for "is this signal active" on the frontend.
 * Always defers to the backend-owned `isActive` field (see
 * backend/internal/domain/signal_status.go's SignalIsActiveAt). Never
 * re-derive activeness from `status`/`type` strings anywhere else in the
 * frontend — every component must go through this helper (or read
 * `signal.isActive` directly) so the Terminal and the Signals page can
 * never disagree about what counts as active.
 */
export function isSignalActive(signal: Pick<Signal, "isActive">): boolean {
  return signal.isActive === true;
}

/** BUY vs SELL, sourced from the backend contract. */
export function signalDirection(signal: Pick<Signal, "direction">): "BUY" | "SELL" {
  return signal.direction;
}

/** Coarse strategy family, sourced from the backend contract. */
export function signalStrategy(signal: Pick<Signal, "strategy">) {
  return signal.strategy;
}

/** Coarse lifecycle bucket (for tabs/sort), sourced from the backend contract. */
export function signalLifecycleGroup(signal: Pick<Signal, "lifecycleGroup">) {
  return signal.lifecycleGroup;
}

export const TERMINAL_STATUSES = new Set([
  "CLOSED",
  "INVALIDATED",
  "EXPIRED",
  "BLOCKED",
  "SUPPRESSED",
]);

/**
 * Human-readable label + color token for a lifecycle group. Centralized so
 * every panel (Terminal, Signals page, pair list badges) renders status
 * consistently instead of each component inventing its own status->color
 * mapping.
 */
export const LIFECYCLE_LABEL: Record<string, { label: string; tone: "positive" | "negative" | "warning" | "muted" | "accent" }> = {
  SETUP: { label: "Setup", tone: "accent" },
  CONFIRMED: { label: "Confirmed", tone: "positive" },
  ACTIVE: { label: "Active", tone: "positive" },
  BLOCKED: { label: "Blocked", tone: "warning" },
  INVALIDATED: { label: "Invalidated", tone: "negative" },
  EXPIRED: { label: "Expired", tone: "muted" },
  CLOSED: { label: "Closed", tone: "muted" },
  WATCH: { label: "Watch", tone: "muted" },
};

export const STRATEGY_LABEL: Record<string, string> = {
  ENTRY_BUY: "Buy Entry",
  PROTECTIVE_SELL: "Protective Sell",
  TAKE_PROFIT: "Take Profit",
  EXIT_WARNING: "Exit Warning",
  AVOID_ENTRY: "Avoid Entry",
};

export function formatPrice(value: number): string {
  if (!Number.isFinite(value)) return "-";
  const maximumFractionDigits = value >= 1000 ? 2 : value >= 1 ? 4 : 8;
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits,
  }).format(value);
}

export function formatCompact(value: number): string {
  if (!Number.isFinite(value)) return "-";
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 2,
  }).format(value);
}

export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return "-";
  return `${value >= 0 ? "+" : ""}${value.toFixed(2)}%`;
}

export function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "short",
    timeStyle: "medium",
  }).format(date);
}

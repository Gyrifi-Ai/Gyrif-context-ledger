const minute = 60;
const hour = 60 * minute;
const day = 24 * hour;

export function formatAge(iso: string, now = Date.now()): string {
  const timestamp = Date.parse(iso);
  if (!Number.isFinite(timestamp)) return "Unknown";
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1_000));
  if (seconds < minute) return "now";
  if (seconds < hour) return `${Math.floor(seconds / minute)}m ago`;
  if (seconds < day) return `${Math.floor(seconds / hour)}h ago`;
  return `${Math.floor(seconds / day)}d ago`;
}
export function Skeleton({ width = "100%", height = "1rem", radius = "var(--radius-sm)", count = 1 }: { width?: string; height?: string; radius?: string; count?: number }) {
  return <div className="grid gap-2" aria-busy="true">{Array.from({ length: count }, (_, index) => <span key={index} className="block animate-pulse bg-muted" style={{ width, height, borderRadius: radius }} />)}</div>;
}

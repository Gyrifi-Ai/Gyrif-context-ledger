import { cn } from "@/lib/utils";

export type StatusTone = "neutral" | "info" | "review" | "success" | "warning" | "danger";

const tones: Record<StatusTone, string> = {
  neutral: "bg-muted text-muted-foreground", info: "bg-info/10 text-info", review: "bg-review/10 text-review",
  success: "bg-success/10 text-success", warning: "bg-warning/10 text-warning", danger: "bg-destructive/10 text-destructive",
};

export function StatusBadge({ label, tone, dot = false }: { label: string; tone: StatusTone; dot?: boolean }) {
  return <span className={cn("inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider", tones[tone])}>{dot && <span className="size-1.5 rounded-full bg-current" />}{label}</span>;
}

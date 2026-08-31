import type { ReactNode } from "react";

export function EmptyState({ icon, title, description, action, variant = "default" }: { icon?: ReactNode; title: string; description?: ReactNode; action?: ReactNode; variant?: "default" | "compact" }) {
  return (
    <div className={`flex flex-col items-center px-6 text-center ${variant === "compact" ? "py-8" : "py-16"}`}>
      {icon && <span className="mb-3 text-muted-foreground">{icon}</span>}
      <strong className="text-base font-semibold">{title}</strong>
      {description && <p className="mt-2 max-w-[44ch] text-sm text-muted-foreground">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

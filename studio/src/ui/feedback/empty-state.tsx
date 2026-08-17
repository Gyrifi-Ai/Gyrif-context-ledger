import type { ReactNode } from "react";

export function EmptyState({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col items-center px-6 py-16 text-center">
      <strong className="text-base font-semibold">{title}</strong>
      <p className="mt-2 max-w-[44ch] text-sm text-muted-foreground">{children}</p>
    </div>
  );
}

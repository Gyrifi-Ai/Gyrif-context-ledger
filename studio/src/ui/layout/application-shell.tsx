import type { ReactNode } from "react";

export function ApplicationShell({ sidebar, topbar, banner, header, children, rail }: { sidebar: ReactNode; topbar: ReactNode; banner?: ReactNode; header: ReactNode; children: ReactNode; rail?: ReactNode }) {
  return <div className="min-h-screen bg-background md:grid md:grid-cols-[var(--shell-sidebar)_minmax(0,1fr)]"><aside className="border-b border-border bg-muted md:sticky md:top-0 md:h-screen md:border-b-0 md:border-r">{sidebar}</aside><div className="min-w-0"><div className="sticky top-0 z-10 flex h-14 items-center border-b border-border bg-card/90 px-4 backdrop-blur-sm md:px-8">{topbar}</div>{banner}<main className="mx-auto max-w-[var(--shell-max)] p-4 md:p-8"><div>{header}</div><div className={rail ? "grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]" : ""}><div>{children}</div>{rail && <aside className="order-first xl:order-last">{rail}</aside>}</div></main></div></div>;
}

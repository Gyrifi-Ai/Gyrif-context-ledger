import { Boxes, GitPullRequest, History, Inbox } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import type { Route } from "../../app/router";
import { api } from "../../api/client";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

const areas: { id: Route; label: string; eyebrow: string; description: string; icon: typeof Boxes }[] = [
  { id: "ledgers", label: "Ledgers", eyebrow: "Ledgers", description: "A ledger is a governed namespace with its own inbox, proposals, and release history.", icon: Boxes },
  { id: "changes", label: "Changes", eyebrow: "Durable inbox", description: "Desired-state mutations waiting to be proposed. Nothing here has touched the target.", icon: Inbox },
  { id: "proposals", label: "Proposals", eyebrow: "Context PRs", description: "Review batched changes, attach evidence, approve, and release.", icon: GitPullRequest },
  { id: "releases", label: "Releases", eyebrow: "Immutable history", description: "Every release was applied to the target and verified before it was recorded.", icon: History },
];

type Health = { state: "connected" | "offline"; version?: string; inference?: string };

function useRuntimeStatus(): Health {
  const [health, setHealth] = useState<Health>({ state: "offline" });
  useEffect(() => {
    let cancelled = false;
    const probe = () =>
      api.status()
        .then((s) => !cancelled && setHealth({ state: "connected", version: s.version, inference: s.inference }))
        .catch(() => !cancelled && setHealth({ state: "offline" }));
    void probe();
    const timer = setInterval(probe, 30_000);
    return () => { cancelled = true; clearInterval(timer); };
  }, []);
  return health;
}

export function ApplicationShell({ route, children, ledgerId }: { route: Route; children: ReactNode; ledgerId: string }) {
  const area = areas.find((entry) => entry.id === route) ?? areas[0];
  const health = useRuntimeStatus();
  return (
    <TooltipProvider delayDuration={200}>
      <div className="grid min-h-screen md:grid-cols-[248px_minmax(0,1fr)]">
        <aside className="flex flex-row items-center gap-4 border-b bg-card/40 px-4 py-3 md:sticky md:top-0 md:h-screen md:flex-col md:items-stretch md:gap-6 md:border-b-0 md:border-r md:px-3 md:py-5">
          <div className="flex items-center gap-3 md:border-b md:border-border/60 md:px-2 md:pb-5">
            <span className="grid h-9 w-9 flex-none place-items-center rounded-lg bg-primary font-bold text-primary-foreground shadow-[0_0_24px_-4px] shadow-primary/50">G</span>
            <div className="hidden md:block">
              <p className="text-sm font-semibold tracking-tight">Gyrifi</p>
              <p className="text-[11px] uppercase tracking-widest text-muted-foreground">Context ledger</p>
            </div>
          </div>
          <nav className="flex gap-1 overflow-x-auto md:grid">
            {areas.map((entry) => (
              <a
                key={entry.id}
                href={`#${entry.id}`}
                aria-current={route === entry.id ? "page" : undefined}
                className={cn(
                  "relative flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
                  route === entry.id && "bg-accent text-accent-foreground before:absolute before:left-0 before:top-1/2 before:hidden before:h-4 before:w-0.5 before:-translate-y-1/2 before:rounded-full before:bg-primary md:before:block",
                )}
              >
                <entry.icon className="h-4 w-4" strokeWidth={1.75} />
                {entry.label}
              </a>
            ))}
          </nav>
          <div className="ml-auto flex items-center md:ml-0 md:mt-auto md:border-t md:border-border/60 md:px-2 md:pt-4">
            <Tooltip>
              <TooltipTrigger className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className={cn("h-2 w-2 rounded-full", health.state === "connected" ? "bg-success shadow-[0_0_8px] shadow-success/60" : "bg-destructive shadow-[0_0_8px] shadow-destructive/60")} />
                {health.state === "connected" ? "Connected" : "Offline"}
              </TooltipTrigger>
              <TooltipContent side="top">
                {health.state === "connected" ? `Runtime ${health.version} · inference ${health.inference}` : "Runtime unreachable"}
              </TooltipContent>
            </Tooltip>
          </div>
        </aside>
        <main className="min-w-0 px-4 py-6 md:px-8 md:py-8">
          <header className="mx-auto mb-6 flex max-w-6xl flex-wrap items-end justify-between gap-4">
            <div>
              <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-muted-foreground">{area.eyebrow}</p>
              <h1 className="text-2xl font-semibold tracking-tight">{area.label}</h1>
              <p className="mt-2 max-w-[68ch] text-sm text-muted-foreground">{area.description}</p>
            </div>
            <code className={cn("max-w-72 truncate rounded-full border bg-background/60 px-3 py-1 text-xs", ledgerId ? "text-muted-foreground" : "italic text-muted-foreground/70")} title={ledgerId || undefined}>
              {ledgerId || "No ledger selected"}
            </code>
          </header>
          <div className="mx-auto max-w-6xl">{children}</div>
        </main>
      </div>
    </TooltipProvider>
  );
}

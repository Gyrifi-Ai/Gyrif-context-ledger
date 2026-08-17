import { useEffect, useState } from "react";
import { api } from "../../api/client";
import type { Release } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function ReleasesPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Release[]>([]);
  const [error, setError] = useState("");
  useEffect(() => { if (ledgerId) api.releases(ledgerId).then((value) => setItems(value.items ?? [])).catch((value: Error) => setError(value.message)); }, [ledgerId]);
  const rollback = async (releaseId: string) => { try { await api.rollback(ledgerId, releaseId); window.location.hash = "proposals"; } catch (value) { setError((value as Error).message); } };
  if (!ledgerId) return <Card><EmptyState title="Select a ledger">Release history is scoped to a ledger.</EmptyState></Card>;
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 border-b">
        <div>
          <CardTitle className="text-base">Release timeline</CardTitle>
          <CardDescription className="mt-1">Applied to the target and verified before recording.</CardDescription>
        </div>
        <span className="text-sm text-muted-foreground">{items.length} releases</span>
      </CardHeader>
      <CardContent className="pt-6">
        {error && <p role="alert" className="mb-4 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 font-mono text-xs text-destructive">{error}</p>}
        {items.length === 0 ? (
          <EmptyState title="No Releases yet">Approved Proposals become immutable Releases here.</EmptyState>
        ) : (
          <ol className="relative ml-2 space-y-8 border-l border-border pb-2">
            {items.map((release, index) => {
              const head = index === 0;
              return (
                <li key={release.id} className="relative pl-7">
                  <span className={cn(
                    "absolute -left-[7px] top-1 h-3.5 w-3.5 rounded-full border-2",
                    head ? "border-primary bg-primary shadow-[0_0_12px] shadow-primary/50" : "border-muted-foreground/50 bg-background",
                  )} />
                  <p className={cn("text-[11px] font-semibold uppercase tracking-widest", head ? "text-success" : "text-muted-foreground")}>{head ? "HEAD" : "History"}</p>
                  <h3 className="mt-1 font-mono text-sm font-medium">{release.id}</h3>
                  <code className="mt-1 block text-xs text-muted-foreground">{release.hash}</code>
                  <p className="mt-1.5 text-sm text-muted-foreground">Proposal {release.proposalId}</p>
                  {!head && <Button variant="destructive" size="sm" className="mt-3" onClick={() => void rollback(release.id)}>Create rollback Proposal</Button>}
                </li>
              );
            })}
          </ol>
        )}
      </CardContent>
    </Card>
  );
}

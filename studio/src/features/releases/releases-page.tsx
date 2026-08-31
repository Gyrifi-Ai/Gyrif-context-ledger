import { useState } from "react";
import { api } from "../../api/client";
import type { Release } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation, useQuery } from "../../app/use-async";
import { AsyncBoundary } from "../../ui/feedback/async-boundary";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function ReleasesPage() {
  const { ledgerId } = useAppState();
  const [lastReleaseId, setLastReleaseId] = useState<string | null>(null);
  const releasesQuery = useQuery("releases", async (signal) => ledgerId ? (await api.releases(ledgerId, { signal })).items ?? [] : [], [ledgerId]);
  const rollbackMutation = useMutation(async (releaseId: string) => {
    await api.rollback(ledgerId, releaseId);
    window.location.hash = "proposals";
  });
  useLedgerEvents(releasesQuery.refetch);
  const rollback = (releaseId: string) => { setLastReleaseId(releaseId); void rollbackMutation.run(releaseId); };
  if (!ledgerId) return <Card><EmptyState title="Select a ledger" description="Release history is scoped to a ledger." /></Card>;
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 border-b">
        <div>
          <CardTitle className="text-base">Release timeline</CardTitle>
          <CardDescription className="mt-1">Applied to the target and verified before recording.</CardDescription>
        </div>
        <span className="text-sm text-muted-foreground">{releasesQuery.data?.length ?? 0} releases</span>
      </CardHeader>
      <CardContent className="pt-6">
        {rollbackMutation.error && <div className="mb-4"><ErrorState title="Unable to create rollback Proposal" message={rollbackMutation.error.message} onRetry={() => { if (lastReleaseId) void rollbackMutation.run(lastReleaseId); }} /></div>}
        <AsyncBoundary query={releasesQuery} empty={<EmptyState title="No Releases yet" description="Approved Proposals become immutable Releases here." />}>
          {(items: Release[]) => (
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
                  {!head && <Button variant="destructive" size="sm" className="mt-3" loading={rollbackMutation.pending} onClick={() => rollback(release.id)}>Create rollback Proposal</Button>}
                </li>
              );
            })}
          </ol>
          )}
        </AsyncBoundary>
      </CardContent>
    </Card>
  );
}

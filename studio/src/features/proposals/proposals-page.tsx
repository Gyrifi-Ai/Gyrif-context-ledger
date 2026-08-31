import { useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation, useQuery } from "../../app/use-async";
import { AsyncBoundary } from "../../ui/feedback/async-boundary";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { proposalTone } from "../shared/status";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

export function ProposalsPage() {
  const { ledgerId } = useAppState();
  const [selected, setSelected] = useState<string[]>([]);
  const [title, setTitle] = useState("");
  const [lastAction, setLastAction] = useState<{ action: "evaluate" | "approve" | "release"; proposal: Proposal } | null>(null);
  const workspaceQuery = useQuery("proposals", async (signal) => {
    if (!ledgerId) return { proposals: [] as Proposal[], changes: [] as Change[] };
    const [proposalResult, changeResult] = await Promise.all([api.proposals(ledgerId, { signal }), api.changes(ledgerId, { signal })]);
    return { proposals: proposalResult.items ?? [], changes: (changeResult.items ?? []).filter((change) => change.status === "READY") };
  }, [ledgerId]);
  const createMutation = useMutation(async ({ proposalTitle, changeIds }: { proposalTitle: string; changeIds: string[] }) => {
    await api.createProposal(ledgerId, { title: proposalTitle, changeIds });
    setTitle("");
    setSelected([]);
    workspaceQuery.refetch();
  });
  const actionMutation = useMutation(async ({ action, proposal }: { action: "evaluate" | "approve" | "release"; proposal: Proposal }) => {
    if (action === "evaluate") await api.evaluate(ledgerId, proposal.id, "The proposed context is accurate, internally consistent, and safe to release.");
    if (action === "approve") await api.approve(ledgerId, proposal.id);
    if (action === "release") await api.release(ledgerId, proposal.id);
    workspaceQuery.refetch();
  });
  useLedgerEvents(ledgerId, workspaceQuery.refetch);
  const create = (event: FormEvent) => { event.preventDefault(); void createMutation.run({ proposalTitle: title, changeIds: selected }); };
  const act = (action: "evaluate" | "approve" | "release", proposal: Proposal) => { const next = { action, proposal }; setLastAction(next); void actionMutation.run(next); };
  if (!ledgerId) return <Card><EmptyState title="Select a ledger" description="Proposals are scoped to a ledger." /></Card>;
  return (
    <div className="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <Card>
        <CardHeader className="border-b">
          <CardTitle className="text-base">Review queue</CardTitle>
          <CardDescription>Evaluate, approve, and release batched changes.</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {actionMutation.error && <div className="p-5"><ErrorState title="Unable to update Proposal" message={actionMutation.error.message} onRetry={() => { if (lastAction) void actionMutation.run(lastAction); }} retryDisabled={actionMutation.blocked} retryTitle={actionMutation.disabledReason} /></div>}
          <AsyncBoundary query={workspaceQuery} empty={<EmptyState title="No Proposals" description="Select Ready Changes to create a reviewable Context PR." />} isEmpty={(workspace) => workspace.proposals.length === 0}>
            {(workspace) => (
            <ul className="divide-y divide-border/60">
              {workspace.proposals.map((proposal) => (
                <li key={proposal.id} className="flex flex-wrap items-center justify-between gap-4 px-5 py-4 transition-colors hover:bg-muted/40">
                  <div className="min-w-0 space-y-1.5">
                    <StatusBadge label={proposal.status} tone={proposalTone(proposal.status)} />
                    <p className="font-medium leading-tight">{proposal.title}</p>
                    <code className="block text-xs text-muted-foreground">{proposal.hash.slice(0, 18)} · {proposal.changeIds.length} changes</code>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button variant="secondary" size="sm" loading={actionMutation.pending} disabled={actionMutation.blocked} title={actionMutation.disabledReason} onClick={() => act("evaluate", proposal)}>Evaluate</Button>
                    <Button variant="secondary" size="sm" loading={actionMutation.pending} disabled={actionMutation.blocked} title={actionMutation.disabledReason} onClick={() => act("approve", proposal)}>Approve</Button>
                    <Button variant="destructive" size="sm" loading={actionMutation.pending} disabled={actionMutation.blocked} title={actionMutation.disabledReason} onClick={() => act("release", proposal)}>Release</Button>
                  </div>
                </li>
              ))}
            </ul>
            )}
          </AsyncBoundary>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="border-b">
          <CardTitle className="text-base">New proposal</CardTitle>
          <CardDescription>Batch Ready Changes into a reviewable unit.</CardDescription>
        </CardHeader>
        <CardContent className="pt-5">
          <form onSubmit={create} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="proposal-title">Title</Label>
              <Input id="proposal-title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="August refund policy refresh" required />
            </div>
            <div className="grid gap-2">
              <Label>Ready changes</Label>
              <div className="grid max-h-44 gap-1 overflow-auto rounded-md border bg-background/60 p-2">
                {(workspaceQuery.data?.changes ?? []).length === 0 && <p className="px-1 py-2 text-xs text-muted-foreground">No Ready Changes in this ledger.</p>}
                {(workspaceQuery.data?.changes ?? []).map((change) => (
                  <label key={change.id} className="flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent">
                    <Checkbox
                      checked={selected.includes(change.id)}
                      onCheckedChange={(checked) => setSelected((old) => checked === true ? [...old, change.id] : old.filter((id) => id !== change.id))}
                    />
                    <span className="truncate">{change.unit}</span>
                  </label>
                ))}
              </div>
            </div>
            {createMutation.error && <ErrorState title="Unable to create Proposal" message={createMutation.error.message} onRetry={() => void createMutation.run({ proposalTitle: title, changeIds: selected })} retryDisabled={createMutation.blocked} retryTitle={createMutation.disabledReason} />}
            <Separator />
            <Button type="submit" className="w-full" disabled={selected.length === 0 || createMutation.blocked} title={createMutation.disabledReason} loading={createMutation.pending}>
              Create Proposal{selected.length > 0 ? ` (${selected.length})` : ""}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

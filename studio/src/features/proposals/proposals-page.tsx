import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

export function ProposalsPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Proposal[]>([]);
  const [changes, setChanges] = useState<Change[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [title, setTitle] = useState("");
  const [error, setError] = useState("");
  const load = async () => { if (!ledgerId) return; try { const [proposalResult, changeResult] = await Promise.all([api.proposals(ledgerId), api.changes(ledgerId)]); setItems(proposalResult.items ?? []); setChanges((changeResult.items ?? []).filter((change) => change.status === "READY")); } catch (value) { setError((value as Error).message); } };
  useEffect(() => { void load(); }, [ledgerId]);
  const create = async (event: FormEvent) => { event.preventDefault(); try { await api.createProposal(ledgerId, { title, changeIds: selected }); setTitle(""); setSelected([]); await load(); } catch (value) { setError((value as Error).message); } };
  const act = async (action: "evaluate" | "approve" | "release", proposal: Proposal) => { try { if (action === "evaluate") await api.evaluate(ledgerId, proposal.id, "The proposed context is accurate, internally consistent, and safe to release."); if (action === "approve") await api.approve(ledgerId, proposal.id); if (action === "release") await api.release(ledgerId, proposal.id); await load(); } catch (value) { setError((value as Error).message); } };
  if (!ledgerId) return <Card><EmptyState title="Select a ledger">Proposals are scoped to a ledger.</EmptyState></Card>;
  return (
    <div className="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <Card>
        <CardHeader className="border-b">
          <CardTitle className="text-base">Review queue</CardTitle>
          <CardDescription>Evaluate, approve, and release batched changes.</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {items.length === 0 ? (
            <EmptyState title="No Proposals">Select Ready Changes to create a reviewable Context PR.</EmptyState>
          ) : (
            <ul className="divide-y divide-border/60">
              {items.map((proposal) => (
                <li key={proposal.id} className="flex flex-wrap items-center justify-between gap-4 px-5 py-4 transition-colors hover:bg-muted/40">
                  <div className="min-w-0 space-y-1.5">
                    <StatusBadge value={proposal.status} />
                    <p className="font-medium leading-tight">{proposal.title}</p>
                    <code className="block text-xs text-muted-foreground">{proposal.hash.slice(0, 18)} · {proposal.changeIds.length} changes</code>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button variant="secondary" size="sm" onClick={() => void act("evaluate", proposal)}>Evaluate</Button>
                    <Button variant="secondary" size="sm" onClick={() => void act("approve", proposal)}>Approve</Button>
                    <Button variant="destructive" size="sm" onClick={() => void act("release", proposal)}>Release</Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
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
                {changes.length === 0 && <p className="px-1 py-2 text-xs text-muted-foreground">No Ready Changes in this ledger.</p>}
                {changes.map((change) => (
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
            {error && <p role="alert" className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 font-mono text-xs text-destructive">{error}</p>}
            <Separator />
            <Button type="submit" className="w-full" disabled={selected.length === 0}>
              Create Proposal{selected.length > 0 ? ` (${selected.length})` : ""}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

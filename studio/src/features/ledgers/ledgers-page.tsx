import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";
import { api } from "../../api/client";
import type { Ledger } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation, useQuery } from "../../app/use-async";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AsyncBoundary } from "../../ui/feedback/async-boundary";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";

export function LedgersPage() {
  const [name, setName] = useState("");
  const [open, setOpen] = useState(false);
  const { ledgerId, refreshLedgers, setLedgerId } = useAppState();
  const ledgerQuery = useQuery("ledgers", async (signal) => (await api.ledgers({ signal })).items ?? [], []);
  const createMutation = useMutation(async ({ name: ledgerName }: { name: string }) => {
    const ledger = await api.createLedger({ name: ledgerName });
    setName("");
    setOpen(false);
    setLedgerId(ledger.id);
    ledgerQuery.refetch();
    void refreshLedgers();
    return ledger;
  });
  useLedgerEvents(ledgerQuery.refetch);
  const create = (event: FormEvent) => {
    event.preventDefault();
    void createMutation.run({ name });
  };
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{ledgerQuery.data?.length ?? 0} {(ledgerQuery.data?.length ?? 0) === 1 ? "ledger" : "ledgers"}</p>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild><Button size="sm"><Plus />New ledger</Button></DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create ledger</DialogTitle>
              <DialogDescription>A ledger governs one Qdrant context collection.</DialogDescription>
            </DialogHeader>
            <form onSubmit={create} className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="ledger-name">Name</Label>
                <Input id="ledger-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="product-docs" required autoFocus />
              </div>
              {createMutation.error && <ErrorState title="Unable to create ledger" message={createMutation.error.message} onRetry={() => void createMutation.run({ name })} retryDisabled={createMutation.blocked} retryTitle={createMutation.disabledReason} />}
              <DialogFooter><Button type="submit" loading={createMutation.pending} disabled={createMutation.blocked} title={createMutation.disabledReason}>Create ledger</Button></DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
      <AsyncBoundary query={ledgerQuery} empty={<Card><EmptyState title="No ledgers yet" description="A ledger is a governed namespace with its own inbox, proposals, and release history. Create the first one to begin." /></Card>}>
        {(items: Ledger[]) => <div className="grid gap-4 sm:grid-cols-2">
          {items.map((ledger) => {
            const active = ledger.id === ledgerId;
            return (
              <Card
                key={ledger.id}
                role="button"
                tabIndex={0}
                onClick={() => setLedgerId(ledger.id)}
                onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); setLedgerId(ledger.id); } }}
                className={`cursor-pointer transition-colors hover:border-ring/60 ${active ? "border-primary/60 bg-primary/[0.04]" : ""}`}
              >
                <CardHeader className="flex-row items-start justify-between space-y-0 p-5 pb-3">
                  <div className="flex items-center gap-3">
                    <span className="grid h-9 w-9 flex-none place-items-center rounded-md bg-success/10 text-sm font-bold text-success">{ledger.name.slice(0, 1).toUpperCase()}</span>
                    <div>
                      <CardTitle className="text-base">{ledger.name}</CardTitle>
                      <CardDescription className="mt-0.5">{ledger.description || "Qdrant context ledger"}</CardDescription>
                    </div>
                  </div>
                  {active && <span className="rounded-full bg-success/15 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider text-success">Active</span>}
                </CardHeader>
                <CardContent className="p-5 pt-0">
                  <code className="text-xs text-muted-foreground">{ledger.id}</code>
                </CardContent>
              </Card>
            );
          })}
        </div>}
      </AsyncBoundary>
    </div>
  );
}

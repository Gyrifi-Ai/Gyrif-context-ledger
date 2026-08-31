import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { api } from "../../api/client";
import type { ReleaseIntent, RetryReleaseIntentResult } from "../../api/types";
import { useMutation } from "../../app/use-async";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "../../ui/primitives/button";
import { Drawer } from "../../ui/layout/drawer";
import { ErrorState } from "../../ui/feedback/error-state";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { formatAge } from "../shared/time";
import { intentTone } from "../shared/status";
import { PlanOperations } from "./plan-drawer";

type RetryResult = { intentId: string; outcome: RetryReleaseIntentResult };

export function RecoveryBanner({ ledgerId, intents, onUpdated, onResolved }: { ledgerId: string; intents: ReleaseIntent[]; onUpdated: () => void; onResolved?: (message: string) => void }) {
  const [open, setOpen] = useState(false);
  const [retryingId, setRetryingId] = useState<string>();
  const [resolvingId, setResolvingId] = useState<string>();
  const [note, setNote] = useState("");
  const retry = useMutation(async (intentId: string): Promise<RetryResult> => ({ intentId, outcome: await api.retryReleaseIntent(ledgerId, intentId) }));
  const resolve = useMutation(async (intentId: string) => { await api.resolveReleaseIntent(ledgerId, intentId, note.trim()); return intentId; });

  useEffect(() => {
    if (!retry.result) return;
    if (retry.result.outcome.resolved) onResolved?.("Release verification succeeded; HEAD advanced.");
    onUpdated();
  }, [retry.result]);

  useEffect(() => {
    if (!resolve.result) return;
    setResolvingId(undefined);
    setNote("");
    onResolved?.("Release Intent marked ABANDONED; HEAD did not move.");
    onUpdated();
  }, [resolve.result]);

  if (intents.length === 0) return null;
  const countLabel = `${intents.length} release ${intents.length === 1 ? "intent" : "intents"} require${intents.length === 1 ? "s" : ""} recovery.`;
  return (
    <>
      <div role="alert" className="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-md border border-warning/50 bg-warning/10 p-4 text-warning">
        <p className="flex items-center gap-2 text-sm font-medium"><AlertTriangle className="size-4" aria-hidden="true" />{countLabel}</p>
        <Button variant="secondary" size="sm" onClick={() => setOpen(true)}>Inspect</Button>
      </div>
      <Drawer open={open} onClose={() => setOpen(false)} title="Release recovery">
        <div className="grid gap-5">
          {intents.map((intent) => {
            const retryResult = retry.result?.intentId === intent.id ? retry.result.outcome : undefined;
            return (
              <section key={intent.id} className="rounded-md border border-border p-3">
                <div className="flex flex-wrap items-center justify-between gap-2"><HashChip value={intent.id} label="Intent" /><StatusBadge label={intent.status} tone={intentTone(intent.status)} dot /></div>
                <p className="mt-2 text-xs text-muted-foreground">Proposal <code>{intent.proposalId}</code> · <time dateTime={intent.createdAt} title={intent.createdAt}>{formatAge(intent.createdAt)}</time></p>
                <details className="mt-3" open><summary className="cursor-pointer text-sm font-medium">Plan · {intent.plan.operations.length} operations</summary><div className="mt-3"><PlanOperations operations={intent.plan.operations} /></div></details>
                {retryResult && !retryResult.resolved && <div className="mt-3 rounded-md border border-warning/40 bg-warning/10 p-3 text-xs text-warning"><p className="font-medium">Verification still disagrees with the target.</p><ul className="mt-2 grid gap-1">{retryResult.mismatches.map((mismatch) => <li key={mismatch.unit}><code>{mismatch.unit}</code>: expected <code>{mismatch.expected || "absent"}</code>, observed <code>{mismatch.observed || "absent"}</code></li>)}</ul></div>}
                {retry.error && retryingId === intent.id && <div className="mt-3"><ErrorState title="Unable to retry verification" message={retry.error.message} onRetry={() => void retry.run(intent.id)} retryDisabled={retry.blocked} retryTitle={retry.disabledReason} /></div>}
                {resolve.error && resolvingId === intent.id && <div className="mt-3"><ErrorState title="Unable to mark Intent resolved" message={resolve.error.message} onRetry={() => void resolve.run(intent.id)} retryDisabled={resolve.blocked || !note.trim()} retryTitle={resolve.disabledReason} /></div>}
                {resolvingId === intent.id && <div className="mt-3 grid gap-2"><label className="text-xs font-medium" htmlFor={`resolution-note-${intent.id}`}>Resolution note</label><Textarea id={`resolution-note-${intent.id}`} value={note} onChange={(event) => setNote(event.target.value)} placeholder="Describe the target inspection and manual repair." /><p className="text-xs text-muted-foreground">This marks the Intent ABANDONED. It does not advance HEAD.</p></div>}
                <div className="mt-3 flex flex-wrap gap-2"><Button size="sm" loading={retry.pending && retryingId === intent.id} disabled={retry.blocked || resolve.pending || retry.pending} title={retry.disabledReason} onClick={() => { setRetryingId(intent.id); retry.reset(); void retry.run(intent.id); }}>Retry verification</Button>{resolvingId === intent.id ? <><Button variant="danger" size="sm" loading={resolve.pending} disabled={!note.trim() || resolve.blocked || retry.pending} title={resolve.disabledReason} onClick={() => void resolve.run(intent.id)}>Mark abandoned</Button><Button variant="ghost" size="sm" onClick={() => { setResolvingId(undefined); setNote(""); resolve.reset(); }}>Cancel</Button></> : <Button variant="secondary" size="sm" disabled={retry.pending || resolve.blocked} title={resolve.disabledReason} onClick={() => { setResolvingId(intent.id); setNote(""); resolve.reset(); }}>Mark resolved</Button>}</div>
              </section>
            );
          })}
        </div>
      </Drawer>
    </>
  );
}
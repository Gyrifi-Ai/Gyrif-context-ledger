import { useEffect, useState } from "react";
import { ApiError, api } from "../../api/client";
import type { ProposalDetail, Release } from "../../api/types";
import { useMutation } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { Button } from "../../ui/primitives/button";
import { ConfirmDialog } from "../../ui/patterns/confirm-dialog";
import { releaseGate } from "./gates";

export function ReleasePanel({ ledgerId, detail, onRefresh }: { ledgerId: string; detail: ProposalDetail; onRefresh: () => void }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const mutation = useMutation(async (_: undefined) => {
    const release = await api.release(ledgerId, detail.proposal.id);
    onRefresh();
    return release;
  });

  useEffect(() => {
    setConfirmOpen(false);
    mutation.reset();
  }, [detail.proposal.id]);

  const release = mutation.result as Release | undefined;
  const unavailable = mutation.error instanceof ApiError && mutation.error.kind === "http" && mutation.error.status === 503 && mutation.error.code === "UNAVAILABLE";
  const released = detail.proposal.status === "RELEASED" || Boolean(release);
  const gate = releaseGate(detail.gates);

  return (
    <div className="grid gap-4">
      {released ? (
        <div className="rounded-md border border-success/30 bg-success/10 p-4 text-sm">
          <p className="font-medium text-success">Release recorded after verification; HEAD has advanced.</p>
          <a className="mt-2 inline-block font-medium text-primary underline" href="#releases">View release</a>
        </div>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">Release mutates the configured Qdrant target only after current evidence and approval are verified by the Runtime.</p>
          {!gate.enabled && gate.reason && <p className="text-sm text-muted-foreground">{gate.reason}</p>}
          <div><Button variant="danger" loading={mutation.pending} disabled={!gate.enabled || mutation.blocked} title={!gate.enabled ? gate.reason : mutation.disabledReason} onClick={() => setConfirmOpen(true)}>Release to Qdrant</Button></div>
        </>
      )}
      {unavailable && (
        <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          <p className="font-medium">{mutation.error?.message}</p>
          <p className="mt-1">The durable Release Intent requires inspection.</p>
          <a className="mt-2 inline-block font-medium underline" href="#releases">Open Releases recovery</a>
        </div>
      )}
      {mutation.error && !unavailable && <ErrorState title="Unable to release Proposal" message={mutation.error.message} onRetry={() => setConfirmOpen(true)} retryDisabled={mutation.blocked || !gate.enabled} retryTitle={!gate.enabled ? gate.reason : mutation.disabledReason} />}
      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title="Release to Qdrant?"
        affectedCount={detail.changes.length}
        confirmLabel="Release to Qdrant"
        consequence={<p>The target collection will be mutated, before-images will be retained, and Ledger HEAD will advance only after verification succeeds.</p>}
        onConfirm={() => { setConfirmOpen(false); void mutation.run(undefined); }}
      />
    </div>
  );
}

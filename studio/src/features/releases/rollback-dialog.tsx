import { useEffect } from "react";
import type { Proposal, Release } from "../../api/types";
import { api } from "../../api/client";
import { useMutation } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { ConfirmDialog } from "../../ui/patterns/confirm-dialog";

export function RollbackDialog({ open, onClose, ledgerId, release, affectedUnitCount, onCreated }: { open: boolean; onClose: () => void; ledgerId: string; release?: Release; affectedUnitCount: number; onCreated: (proposal: Proposal) => void }) {
  const mutation = useMutation(async (releaseId: string) => api.rollback(ledgerId, releaseId));

  useEffect(() => {
    mutation.reset();
  }, [release?.id]);

  useEffect(() => {
    if (!mutation.result) return;
    onCreated(mutation.result);
    onClose();
  }, [mutation.result]);

  return (
    <ConfirmDialog
      open={open}
      onClose={onClose}
      title="Create rollback proposal?"
      affectedCount={affectedUnitCount}
      confirmLabel="Create rollback proposal"
      confirmLoading={mutation.pending}
      confirmDisabled={!release || mutation.blocked}
      confirmTitle={mutation.disabledReason}
      consequence={<div className="grid gap-3"><p>This creates a <strong className="text-foreground">new proposal</strong>; it does not rewind history.</p><p><strong className="text-foreground">{affectedUnitCount} units</strong> will be restored to their state at this release.</p><p>The proposal must be evaluated, approved, and released like any other.</p><p><strong className="text-foreground">HEAD will move forward</strong> to a new release after verification.</p>{mutation.error && <ErrorState title="Unable to create rollback proposal" message={mutation.error.message} onRetry={() => release && void mutation.run(release.id)} retryDisabled={mutation.blocked} retryTitle={mutation.disabledReason} />}</div>}
      onConfirm={() => release && void mutation.run(release.id)}
    />
  );
}
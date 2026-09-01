import { useEffect, useState } from "react";
import type { Change } from "../../api/types";
import { Button } from "../../ui/primitives/button";
import { Field } from "../../ui/primitives/field";
import { Textarea } from "../../ui/primitives/textarea";
import { Drawer } from "../../ui/layout/drawer";
import { ConfirmDialog } from "../../ui/patterns/confirm-dialog";
import { CodeBlock } from "../../ui/patterns/code-block";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { Skeleton } from "../../ui/feedback/skeleton";
import { changeTone } from "../shared/status";

export function ChangeDetailDrawer({ change, onClose, onWithdraw, withdrawPending = false, withdrawBlocked = false, withdrawDisabledReason, withdrawError }: { change: Change | null; onClose: () => void; onWithdraw: (reason: string) => void; withdrawPending?: boolean; withdrawBlocked?: boolean; withdrawDisabledReason?: string; withdrawError?: string }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [reason, setReason] = useState("");
  useEffect(() => {
    setConfirmOpen(false);
    setReason("");
  }, [change?.id, change?.status]);
  const withdrawable = change?.status === "ACCEPTED" || change?.status === "READY" || change?.status === "INVALID";
  return (
    <>
      <Drawer open={change !== null} onClose={onClose} title="Change detail" footer={withdrawable ? <div className="flex justify-end"><Button variant="danger" onClick={() => setConfirmOpen(true)}>Withdraw</Button></div> : undefined}>
        {change && (
        <div className="grid gap-5 text-sm">
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-3">
            <dt className="text-muted-foreground">ID</dt><dd className="break-all font-mono text-xs">{change.id}</dd>
            <dt className="text-muted-foreground">Sequence</dt><dd>{change.sequence}</dd>
            <dt className="text-muted-foreground">Unit</dt><dd className="break-all font-mono text-xs">{change.unit}</dd>
            <dt className="text-muted-foreground">Action</dt><dd>{change.action}</dd>
            <dt className="text-muted-foreground">Status</dt><dd><StatusBadge label={change.stalled ? "Stalled" : change.status === "ACCEPTED" ? "Preparing" : change.status} tone={changeTone(change.status)} /></dd>
            <dt className="text-muted-foreground">Created</dt><dd><time dateTime={change.createdAt}>{change.createdAt}</time></dd>
          </dl>
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Base fingerprint</p>{change.status === "ACCEPTED" ? <div aria-label="Preparing base fingerprint"><Skeleton width="12rem" /></div> : change.baseFingerprint ? <HashChip value={change.baseFingerprint} /> : <p>Unit absent when prepared.</p>}</div>
          {change.status === "INVALID" && <div role="alert" className="rounded-md border border-danger/30 bg-danger/5 p-3"><p className="font-medium text-danger">Invalid Change</p><p className="mt-1">{change.invalidReason}</p></div>}
          {change.noop && <p className="rounded-md border border-border bg-muted/40 p-3 font-medium">No change needed</p>}
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Desired fingerprint</p><HashChip value={change.desiredFingerprint} /></div>
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Desired value</p><CodeBlock value={change.desired ?? null} /></div>
        </div>
        )}
      </Drawer>
      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title="Withdraw Change?"
        consequence={<div className="grid gap-3"><p>The Change remains in the audit trail but cannot be proposed.</p><Field label="Reason" hint="Required and recorded with the withdrawal."><Textarea aria-label="Withdrawal reason" value={reason} onChange={(event) => setReason(event.target.value)} rows={3} required /></Field>{withdrawError && <p role="alert" className="text-danger">{withdrawError}</p>}</div>}
        affectedCount={1}
        confirmLabel="Withdraw Change"
        confirmLoading={withdrawPending}
        confirmDisabled={!reason.trim() || withdrawBlocked}
        confirmTitle={!reason.trim() ? "A withdrawal reason is required." : withdrawDisabledReason}
        onConfirm={() => onWithdraw(reason.trim())}
      />
    </>
  );
}
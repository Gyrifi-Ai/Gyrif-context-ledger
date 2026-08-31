import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { ActionGate, Approval, Proposal } from "../../api/types";
import { useMutation } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { Button } from "../../ui/primitives/button";
import { Field } from "../../ui/primitives/field";
import { Input } from "../../ui/primitives/input";
import { HashChip } from "../../ui/patterns/hash-chip";

const actorStorageKey = "gyrifi.approvalActor";

function storedActor(): string {
  return typeof localStorage === "undefined" || typeof localStorage.getItem !== "function" ? "local-user" : localStorage.getItem(actorStorageKey) ?? "local-user";
}

export function ApprovalPanel({ ledgerId, proposal, approval, gate, onRefresh }: { ledgerId: string; proposal: Proposal; approval?: Approval; gate: ActionGate; onRefresh: () => void }) {
  const [actor, setActor] = useState(storedActor);
  const mutation = useMutation(async (value: string) => {
    await api.approve(ledgerId, proposal.id, value);
    if (typeof localStorage !== "undefined" && typeof localStorage.setItem === "function") localStorage.setItem(actorStorageKey, value);
    onRefresh();
  });

  useEffect(() => mutation.reset(), [proposal.id]);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (actor.trim() && gate.enabled) void mutation.run(actor.trim());
  };

  return (
    <div className="grid gap-4">
      {approval ? (
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Actor</dt><dd>{approval.actor}</dd>
          <dt className="text-muted-foreground">Approved</dt><dd><time dateTime={approval.createdAt}>{approval.createdAt}</time></dd>
          <dt className="text-muted-foreground">Bound hash</dt><dd><HashChip value={approval.proposalHash} /></dd>
        </dl>
      ) : <p className="text-sm text-muted-foreground">No current approval has been recorded.</p>}
      {!approval && (
        <form className="grid gap-3" onSubmit={submit}>
          <Field label="Approving actor" hint="Saved locally as the default for the next approval.">
            <Input value={actor} onChange={(event) => { setActor(event.target.value); mutation.reset(); }} required />
          </Field>
          {!gate.enabled && gate.reason && <p className="text-sm text-muted-foreground">{gate.reason}</p>}
          <div><Button type="submit" loading={mutation.pending} disabled={!actor.trim() || !gate.enabled || mutation.blocked} title={!gate.enabled ? gate.reason : mutation.disabledReason}>Approve</Button></div>
        </form>
      )}
      {mutation.error && <ErrorState title="Unable to approve Proposal" message={mutation.error.message} onRetry={() => void mutation.run(actor.trim())} retryDisabled={mutation.blocked || !gate.enabled || !actor.trim()} retryTitle={!gate.enabled ? gate.reason : mutation.disabledReason} />}
    </div>
  );
}

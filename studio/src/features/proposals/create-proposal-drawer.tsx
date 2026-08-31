import { useEffect, useState, type FormEvent } from "react";
import { ArrowDown, ArrowUp } from "lucide-react";
import { ApiError, api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useMutation } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { Button } from "../../ui/primitives/button";
import { Field } from "../../ui/primitives/field";
import { Input } from "../../ui/primitives/input";
import { Drawer } from "../../ui/layout/drawer";
import { DataTable, type Column } from "../../ui/patterns/data-table";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { changeTone } from "../shared/status";
import { moveOrdered } from "./proposal-view";

const columns: Column<Change>[] = [
  { key: "sequence", header: "SEQ", align: "end", render: (change) => change.sequence },
  { key: "unit", header: "UNIT", mono: true, render: (change) => change.unit },
  { key: "action", header: "ACTION", render: (change) => change.action },
  { key: "fingerprint", header: "DESIRED FINGERPRINT", render: (change) => <span onClick={(event) => event.stopPropagation()}><HashChip value={change.desiredFingerprint} /></span> },
  { key: "status", header: "STATUS", render: (change) => <StatusBadge label={change.status} tone={changeTone(change.status)} /> },
];

export function ProposalOrder({ changes, orderedIds, onChange }: { changes: Change[]; orderedIds: string[]; onChange: (ids: string[]) => void }) {
  if (changes.length === 0) return null;
  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">Application order</h3>
      <ol className="grid gap-2">
        {changes.map((change, index) => (
          <li key={change.id} className="flex items-center gap-2 rounded-md border border-border bg-muted p-3">
            <span className="min-w-0 flex-1"><span className="block truncate font-mono text-xs">{index + 1}. {change.unit}</span><span className="text-xs text-muted-foreground">{change.action}</span></span>
            <Button variant="ghost" size="sm" aria-label={`Move ${change.unit} up`} disabled={index === 0} onClick={() => onChange(moveOrdered(orderedIds, index, -1))}><ArrowUp className="size-4" aria-hidden="true" /></Button>
            <Button variant="ghost" size="sm" aria-label={`Move ${change.unit} down`} disabled={index === changes.length - 1} onClick={() => onChange(moveOrdered(orderedIds, index, 1))}><ArrowDown className="size-4" aria-hidden="true" /></Button>
          </li>
        ))}
      </ol>
    </div>
  );
}

export function CreateProposalDrawer({ open, ledgerId, changes, hasMoreChanges = false, loadingMoreChanges = false, onLoadMoreChanges, onClose, onCreated, onConflict }: { open: boolean; ledgerId: string; changes: Change[]; hasMoreChanges?: boolean; loadingMoreChanges?: boolean; onLoadMoreChanges?: () => void; onClose: () => void; onCreated: (proposal: Proposal) => void; onConflict: () => void }) {
  const [title, setTitle] = useState("");
  const [orderedIds, setOrderedIds] = useState<string[]>([]);
  const mutation = useMutation(async ({ proposalTitle, changeIds }: { proposalTitle: string; changeIds: string[] }) => {
    try {
      const proposal = await api.createProposal(ledgerId, { title: proposalTitle, changeIds });
      onCreated(proposal);
      return proposal;
    } catch (error) {
      if (error instanceof ApiError && error.code === "CONFLICT") onConflict();
      throw error;
    }
  });

  useEffect(() => {
    if (!open) return;
    setTitle("");
    setOrderedIds([]);
    mutation.reset();
  }, [open]);

  useEffect(() => {
    const ready = new Set(changes.map((change) => change.id));
    setOrderedIds((ids) => ids.filter((id) => ready.has(id)));
  }, [changes]);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (title.trim() && orderedIds.length > 0) void mutation.run({ proposalTitle: title.trim(), changeIds: orderedIds });
  };
  const orderedChanges = orderedIds.map((id) => changes.find((change) => change.id === id)).filter((change): change is Change => Boolean(change));

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="New proposal"
      footer={<div className="flex justify-end gap-3"><Button variant="secondary" onClick={onClose}>Cancel</Button><Button type="submit" form="create-proposal-workspace-form" loading={mutation.pending} disabled={!title.trim() || orderedIds.length === 0 || mutation.blocked} title={mutation.disabledReason}>Create proposal</Button></div>}
    >
      <form id="create-proposal-workspace-form" className="grid gap-5" onSubmit={submit}>
        <Field label="Title" hint="Required; describes this ordered immutable change set."><Input value={title} onChange={(event) => { setTitle(event.target.value); mutation.reset(); }} required placeholder="August refund policy refresh" /></Field>
        <div>
          <h3 className="text-sm font-semibold">Ready Changes</h3>
          <p className="mb-3 mt-1 text-xs text-muted-foreground">Selection order affects the Proposal hash. Reorder the selected Changes below before creation.</p>
          <div className="max-h-80 overflow-auto rounded-md border border-border">
            <DataTable columns={columns} rows={changes} getRowId={(change) => change.id} selectable selectedIds={orderedIds} onSelectionChange={setOrderedIds} empty={<p className="p-4 text-sm text-muted-foreground">No READY Changes are available.</p>} />
          </div>
          {hasMoreChanges && <div className="mt-3 text-center"><Button variant="secondary" size="sm" loading={loadingMoreChanges} disabled={loadingMoreChanges} onClick={onLoadMoreChanges}>Load more Changes</Button></div>}
        </div>
        <ProposalOrder changes={orderedChanges} orderedIds={orderedIds} onChange={setOrderedIds} />
        {mutation.error && <ErrorState title="Unable to create Proposal" message={mutation.error.message} onRetry={() => void mutation.run({ proposalTitle: title.trim(), changeIds: orderedIds })} retryDisabled={mutation.blocked || !title.trim() || orderedIds.length === 0} retryTitle={mutation.disabledReason} />}
      </form>
    </Drawer>
  );
}

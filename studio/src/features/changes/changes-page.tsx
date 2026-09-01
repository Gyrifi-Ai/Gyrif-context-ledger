import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { ArrowDown, ArrowUp, Plus } from "lucide-react";
import { ApiError, api } from "../../api/client";
import type { Change } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation } from "../../app/use-async";
import { usePaginatedQuery, type PaginatedQueryResult } from "../../app/use-paginated-query";
import { Button } from "../../ui/primitives/button";
import { Field, FieldGroup } from "../../ui/primitives/field";
import { Input } from "../../ui/primitives/input";
import { Segmented } from "../../ui/primitives/segmented";
import { Textarea } from "../../ui/primitives/textarea";
import { Drawer } from "../../ui/layout/drawer";
import { PageHeader } from "../../ui/layout/page-header";
import { Panel } from "../../ui/layout/panel";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { CodeBlock } from "../../ui/patterns/code-block";
import { DataTable, type Column } from "../../ui/patterns/data-table";
import { HashChip } from "../../ui/patterns/hash-chip";
import { Stat } from "../../ui/patterns/stat";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { changeTone } from "../shared/status";
import { formatAge } from "../shared/time";
import { countChangeStatuses, filterChangesByUnit, moveOrdered, newIdempotencyKey, prepareChangeSubmission, validateDesiredJson, type ChangeFilters, type ChangeSubmission } from "./changes-page.logic";
import { SelectionActionBar } from "./selection-action-bar";
import { ChangeDetailDrawer } from "./change-detail-drawer";

const confirmationDuration = 3_000;
const statusOptions: ChangeFilters["status"][] = ["ALL", "ACCEPTED", "READY", "INVALID", "RELEASED", "WITHDRAWN"];
const actionOptions: ChangeFilters["action"][] = ["ALL", "PUT", "DELETE"];

const columns: Column<Change>[] = [
  { key: "sequence", header: "SEQ", align: "end", render: (change) => change.sequence },
  { key: "unit", header: "UNIT", mono: true, render: (change) => change.unit },
  { key: "action", header: "ACTION", render: (change) => change.action },
  {
    key: "fingerprint",
    header: "DESIRED FINGERPRINT",
    render: (change) => <span onClick={(event) => event.stopPropagation()}><HashChip value={change.desiredFingerprint} /></span>,
  },
  { key: "status", header: "STATUS", render: (change) => <div className="flex flex-wrap items-center gap-2"><StatusBadge label={change.stalled ? "Stalled" : change.status === "ACCEPTED" ? "Preparing" : change.status} tone={changeTone(change.status)} />{change.noop && <span className="text-xs font-medium text-muted-foreground">No change needed</span>}</div> },
  { key: "age", header: "AGE", render: (change) => <time dateTime={change.createdAt} title={change.createdAt}>{formatAge(change.createdAt)}</time> },
];

function ChangesSkeleton() {
  return <Panel padding="none"><DataTable loading selectable columns={columns} rows={[]} getRowId={(change) => change.id} /></Panel>;
}

function ChangesSurface({ query, filtered, selectedIds, highlightedId, onSelectionChange, onRowClick, onSubmit }: {
  query: PaginatedQueryResult<Change>;
  filtered: Change[];
  selectedIds: string[];
  highlightedId: string;
  onSelectionChange: (ids: string[]) => void;
  onRowClick: (change: Change) => void;
  onSubmit: () => void;
}) {
  if (query.loading && query.data === undefined) return <ChangesSkeleton />;
  if (query.error) return <ErrorState title="Unable to load Changes" message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><ChangesSkeleton /></div>;
  if (query.data.length === 0) {
    return (
      <Panel>
        <EmptyState title="No Changes yet" description="Applications submit desired-state mutations to the versioned API; nothing touches the target until a verified Release." action={<Button onClick={onSubmit}>Submit your first Change</Button>} />
        <div className="mx-auto mt-4 max-w-3xl"><CodeBlock language="shell" value={`curl -X POST "/api/v1/ledgers/{id}/changes" -H "Content-Type: application/json" -d '{"unit":"point/42","action":"PUT","desired":{"id":42},"idempotencyKey":"request-42"}'`} /></div>
      </Panel>
    );
  }
  return (
    <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>
      <Panel padding="none">
        <DataTable
          columns={columns}
          rows={filtered}
          getRowId={(change) => change.id}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={onSelectionChange}
          isRowSelectable={(change) => change.status === "READY"}
          getSelectionDisabledReason={(change) => `${change.status} Changes cannot be added to a Proposal.`}
          highlightedId={highlightedId}
          onRowClick={onRowClick}
          empty={<EmptyState variant="compact" title="No Changes match these filters" description="Clear or change a filter to see more of the durable inbox." />}
        />
      </Panel>
      {query.nextCursor && <div className="mt-5 flex justify-center"><Button variant="secondary" loading={query.loadingMore} disabled={query.loadingMore || query.refetching} onClick={query.loadMore}>Load more</Button></div>}
      {query.loadMoreError && <div className="mt-4"><ErrorState title="Unable to load more Changes" message={query.loadMoreError.message} onRetry={query.loadMore} retryDisabled={query.loadingMore} /></div>}
    </div>
  );
}

export function ChangesPage() {
  const { ledgerId, openLedgerSwitcher } = useAppState();
  const [filters, setFilters] = useState<ChangeFilters>({ status: "ALL", action: "ALL", unit: "" });
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [detail, setDetail] = useState<Change | null>(null);
  const [submitOpen, setSubmitOpen] = useState(false);
  const [proposalOpen, setProposalOpen] = useState(false);
  const [proposalTitle, setProposalTitle] = useState("");
  const [proposalOrder, setProposalOrder] = useState<string[]>([]);
  const [unit, setUnit] = useState("");
  const [action, setAction] = useState<Change["action"]>("PUT");
  const [desired, setDesired] = useState("{}");
  const [idempotencyKey, setIdempotencyKey] = useState("");
  const [jsonTouched, setJsonTouched] = useState(false);
  const [highlightedId, setHighlightedId] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const confirmationTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const submitButton = useRef<HTMLButtonElement>(null);
  const unitInput = useRef<HTMLInputElement>(null);

  const changesQuery = usePaginatedQuery(
    "changes",
    (cursor, signal) => ledgerId
      ? api.changes(ledgerId, { cursor, status: filters.status === "ALL" ? undefined : filters.status, action: filters.action === "ALL" ? undefined : filters.action }, { signal })
      : Promise.resolve({ items: [] }),
    [ledgerId, filters.status, filters.action],
  );
  useLedgerEvents(ledgerId, changesQuery.refetch);

  const showConfirmation = (message: string, changeId = "") => {
    if (confirmationTimer.current) clearTimeout(confirmationTimer.current);
    setConfirmation(message);
    setHighlightedId(changeId);
    confirmationTimer.current = setTimeout(() => {
      setConfirmation("");
      setHighlightedId("");
    }, confirmationDuration);
  };

  const submitMutation = useMutation(async (input: ChangeSubmission) => {
    const change = await api.createChange(ledgerId, input);
    setSubmitOpen(false);
    changesQuery.refetch();
    showConfirmation(`Change ${change.id} accepted into the durable inbox.`, change.id);
    return change;
  });
  const proposalMutation = useMutation(async ({ title, changeIds }: { title: string; changeIds: string[] }) => {
    try {
      const proposal = await api.createProposal(ledgerId, { title, changeIds });
      setSelectedIds([]);
      setProposalOpen(false);
      window.location.hash = "proposals";
      return proposal;
    } catch (error) {
      if (error instanceof ApiError && error.code === "CONFLICT") changesQuery.refetch();
      throw error;
    }
  });
  const withdrawMutation = useMutation(async ({ change, reason }: { change: Change; reason: string }) => {
    await api.withdrawChange(ledgerId, change.id, reason);
    setDetail({ ...change, status: "WITHDRAWN" });
    changesQuery.refetch();
    showConfirmation(`Change ${change.id} withdrawn.`, change.id);
  });

  useEffect(() => () => {
    if (confirmationTimer.current) clearTimeout(confirmationTimer.current);
  }, []);

  useEffect(() => {
    const ready = new Set((changesQuery.data ?? []).filter((change) => change.status === "READY").map((change) => change.id));
    setSelectedIds((ids) => ids.filter((id) => ready.has(id)));
  }, [changesQuery.data]);

  useEffect(() => {
    if (submitOpen) unitInput.current?.focus();
    if (!submitOpen && submitMutation.result) submitButton.current?.focus();
  }, [submitOpen, submitMutation.result]);

  const items = changesQuery.data ?? [];
  const filtered = useMemo(() => filterChangesByUnit(items, filters.unit), [items, filters.unit]);
  const counts = useMemo(() => countChangeStatuses(items), [items]);
  const jsonError = action === "PUT" ? validateDesiredJson(desired) : undefined;
  const idempotencyConflict = submitMutation.error instanceof ApiError && submitMutation.error.code === "CONFLICT" ? submitMutation.error.message : undefined;
  const submitError = submitMutation.error && !idempotencyConflict ? submitMutation.error : undefined;

  const openSubmit = () => {
    setUnit("");
    setAction("PUT");
    setDesired("{}");
    setIdempotencyKey(newIdempotencyKey());
    setJsonTouched(false);
    submitMutation.reset();
    setSubmitOpen(true);
  };

  const attemptSubmit = () => {
    setJsonTouched(true);
    const prepared = prepareChangeSubmission(unit, action, desired, idempotencyKey);
    if (!prepared.input) return;
    void submitMutation.run(prepared.input);
  };

  const submitChange = (event: FormEvent) => {
    event.preventDefault();
    attemptSubmit();
  };

  const formatDesired = () => {
    setJsonTouched(true);
    if (jsonError) return;
    setDesired(JSON.stringify(JSON.parse(desired), null, 2));
  };

  const openProposal = () => {
    setProposalTitle("");
    setProposalOrder([...selectedIds]);
    proposalMutation.reset();
    setProposalOpen(true);
  };

  const createProposal = (event: FormEvent) => {
    event.preventDefault();
    if (!proposalTitle.trim() || proposalOrder.length === 0) return;
    void proposalMutation.run({ title: proposalTitle.trim(), changeIds: proposalOrder });
  };

  if (!ledgerId) {
    return (
      <>
        <PageHeader eyebrow="DURABLE INBOX" title="Changes" description="Desired-state mutations waiting to be proposed. Nothing here has touched the target." />
        <Panel><EmptyState title="Select a ledger" description="Choose a governed namespace before reviewing or submitting Changes." action={<Button onClick={openLedgerSwitcher}>Select ledger</Button>} /></Panel>
      </>
    );
  }

  const orderedChanges = proposalOrder.map((id) => items.find((change) => change.id === id)).filter((change): change is Change => Boolean(change));

  return (
    <>
      <PageHeader eyebrow="DURABLE INBOX" title="Changes" description="Desired-state mutations waiting to be proposed. Nothing here has touched the target." actions={<Button ref={submitButton} size="sm" iconLeft={<Plus className="size-4" aria-hidden="true" />} onClick={openSubmit}>Submit change</Button>} />
      {confirmation && <p role="status" className="mb-4 text-sm font-medium text-success">{confirmation}</p>}
      <div className="mb-5 grid grid-cols-1 gap-4 rounded-lg border border-border bg-card p-4 shadow-sm sm:grid-cols-3">
        <Stat label="Ready" value={counts.ready} />
        <Stat label="Released" value={counts.released} />
        <Stat label="Invalid" value={counts.invalid} tone={counts.invalid > 0 ? "danger" : "default"} />
      </div>
      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border border-border bg-card p-4 shadow-sm">
        <label className="grid gap-1 text-xs font-medium text-muted-foreground">Status
          <select aria-label="Filter by status" value={filters.status} onChange={(event) => setFilters((value) => ({ ...value, status: event.target.value as ChangeFilters["status"] }))} className="h-9 rounded-sm border border-input bg-muted px-3 text-sm text-foreground">
            {statusOptions.map((value) => <option key={value} value={value}>{value === "ALL" ? "All statuses" : value}</option>)}
          </select>
        </label>
        <label className="grid gap-1 text-xs font-medium text-muted-foreground">Action
          <select aria-label="Filter by action" value={filters.action} onChange={(event) => setFilters((value) => ({ ...value, action: event.target.value as ChangeFilters["action"] }))} className="h-9 rounded-sm border border-input bg-muted px-3 text-sm text-foreground">
            {actionOptions.map((value) => <option key={value} value={value}>{value === "ALL" ? "All actions" : value}</option>)}
          </select>
        </label>
        <label className="grid min-w-60 flex-1 gap-1 text-xs font-medium text-muted-foreground">Unit
          <Input aria-label="Search units" value={filters.unit} onChange={(event) => setFilters((value) => ({ ...value, unit: event.target.value }))} placeholder="Search unit…" />
        </label>
        <p className="w-full text-xs text-muted-foreground">Status and action filter the complete server history. Unit search matches the rows loaded so far.</p>
      </div>
      <ChangesSurface query={changesQuery} filtered={filtered} selectedIds={selectedIds} highlightedId={highlightedId} onSelectionChange={setSelectedIds} onRowClick={setDetail} onSubmit={openSubmit} />

      <SelectionActionBar count={selectedIds.length} onClear={() => setSelectedIds([])} onCreateProposal={openProposal} />

      <ChangeDetailDrawer change={detail} onClose={() => setDetail(null)} onWithdraw={(reason) => { if (detail) void withdrawMutation.run({ change: detail, reason }); }} withdrawPending={withdrawMutation.pending} withdrawBlocked={withdrawMutation.blocked} withdrawDisabledReason={withdrawMutation.disabledReason} withdrawError={withdrawMutation.error?.message} />

      <Drawer
        open={submitOpen}
        onClose={() => setSubmitOpen(false)}
        title="Submit change"
        footer={<div className="flex justify-end gap-3"><Button variant="secondary" onClick={() => setSubmitOpen(false)}>Cancel</Button><Button type="submit" form="submit-change-form" loading={submitMutation.pending} disabled={submitMutation.blocked} title={submitMutation.disabledReason}>Submit change</Button></div>}
      >
        <form id="submit-change-form" onSubmit={submitChange}>
          <p className="mb-5 text-sm text-muted-foreground">Accepted into the durable inbox; this does not touch the target.</p>
          <FieldGroup>
            <Field label="Unit" hint="Required logical unit key."><Input ref={unitInput} value={unit} onChange={(event) => { setUnit(event.target.value); submitMutation.reset(); }} placeholder="point/42" required /></Field>
            <div className="grid gap-2"><span className="text-sm font-medium text-secondary">Action</span><Segmented value={action} onChange={(value) => { setAction(value as Change["action"]); submitMutation.reset(); }} options={[{ value: "PUT", label: "PUT" }, { value: "DELETE", label: "DELETE" }]} /></div>
            {action === "PUT" && <Field label="Desired JSON" hint="A complete desired JSON value." error={jsonTouched ? jsonError : undefined}><Textarea value={desired} onChange={(event) => { setDesired(event.target.value); submitMutation.reset(); }} onBlur={() => setJsonTouched(true)} rows={10} spellCheck={false} /></Field>}
            {action === "PUT" && <Button variant="secondary" size="sm" onClick={formatDesired}>Format JSON</Button>}
            <Field label="Idempotency key" hint="Visible and editable; reuse only for the same logical request." error={idempotencyConflict}><Input value={idempotencyKey} onChange={(event) => { setIdempotencyKey(event.target.value); submitMutation.reset(); }} required /></Field>
          </FieldGroup>
          {submitError && <div className="mt-5"><ErrorState title="Unable to submit Change" message={submitError.message} onRetry={attemptSubmit} retryDisabled={submitMutation.blocked} retryTitle={submitMutation.disabledReason} /></div>}
        </form>
      </Drawer>

      <Drawer
        open={proposalOpen}
        onClose={() => setProposalOpen(false)}
        title="Create proposal"
        footer={<div className="flex justify-end gap-3"><Button variant="secondary" onClick={() => setProposalOpen(false)}>Cancel</Button><Button type="submit" form="create-proposal-form" loading={proposalMutation.pending} disabled={proposalMutation.blocked || proposalOrder.length === 0} title={proposalMutation.disabledReason}>Create proposal</Button></div>}
      >
        <form id="create-proposal-form" onSubmit={createProposal}>
          <Field label="Title" hint="Required; describes the ordered change set."><Input value={proposalTitle} onChange={(event) => { setProposalTitle(event.target.value); proposalMutation.reset(); }} placeholder="Update support context" required /></Field>
          <div className="mt-5">
            <p className="mb-2 text-sm font-medium">Ordered Changes</p>
            <p className="mb-3 text-xs text-muted-foreground">Order changes the Proposal hash. Use the controls to set the intended application sequence.</p>
            <ol className="grid gap-2">
              {orderedChanges.map((change, index) => (
                <li key={change.id} className="flex items-center gap-2 rounded-md border border-border bg-muted p-3">
                  <span className="min-w-0 flex-1"><span className="block truncate font-mono text-xs">{change.unit}</span><span className="text-xs text-muted-foreground">{change.action} · sequence {change.sequence}</span></span>
                  <Button variant="ghost" size="sm" aria-label={`Move ${change.unit} up`} disabled={index === 0} onClick={() => setProposalOrder(moveOrdered(proposalOrder, index, -1))}><ArrowUp className="size-4" aria-hidden="true" /></Button>
                  <Button variant="ghost" size="sm" aria-label={`Move ${change.unit} down`} disabled={index === orderedChanges.length - 1} onClick={() => setProposalOrder(moveOrdered(proposalOrder, index, 1))}><ArrowDown className="size-4" aria-hidden="true" /></Button>
                </li>
              ))}
            </ol>
          </div>
          {proposalMutation.error && <div className="mt-5"><ErrorState title="Unable to create Proposal" message={proposalMutation.error.message} onRetry={() => void proposalMutation.run({ title: proposalTitle.trim(), changeIds: proposalOrder })} retryDisabled={proposalMutation.blocked} retryTitle={proposalMutation.disabledReason} /></div>}
        </form>
      </Drawer>
    </>
  );
}

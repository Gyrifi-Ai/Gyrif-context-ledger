import { useEffect, useRef, useState, type FormEvent } from "react";
import { Plus } from "lucide-react";
import { ApiError, api } from "../../api/client";
import type { Ledger } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation, useQuery } from "../../app/use-async";
import { usePaginatedQuery, type PaginatedQueryResult } from "../../app/use-paginated-query";
import { Button } from "../../ui/primitives/button";
import { Field, FieldGroup } from "../../ui/primitives/field";
import { Input } from "../../ui/primitives/input";
import { Textarea } from "../../ui/primitives/textarea";
import { Drawer } from "../../ui/layout/drawer";
import { PageHeader } from "../../ui/layout/page-header";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { Skeleton } from "../../ui/feedback/skeleton";
import { ConfirmDialog } from "../../ui/patterns/confirm-dialog";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { cn } from "../../lib/utils";
import { countReadyChanges, ledgerDescriptionMaxLength, ledgerNameMaxLength, validateLedgerForm } from "./ledgers-page.logic";

const confirmationDurationSeconds = 3;
const millisecondsPerSecond = 1_000;

function LedgerCard({ ledger, active, onSelect, onLifecycle }: { ledger: Ledger; active: boolean; onSelect: (ledger: Ledger) => void; onLifecycle: (ledger: Ledger) => void }) {
  const readyQuery = useQuery(
    `ledger-ready-count-${ledger.id}`,
    async (signal) => {
      const page = await api.changes(ledger.id, { limit: 200, status: "READY" }, { signal });
      return page.nextCursor ? `${countReadyChanges(page.items ?? [])}+` : countReadyChanges(page.items ?? []);
    },
    [ledger.id],
  );
  const releasesQuery = useQuery(
    `ledger-release-count-${ledger.id}`,
    async (signal) => {
      const page = await api.releases(ledger.id, { limit: 200 }, { signal });
      return page.nextCursor ? `${page.items?.length ?? 0}+` : page.items?.length ?? 0;
    },
    [ledger.id],
  );
  useLedgerEvents(ledger.id, readyQuery.refetch);
  useLedgerEvents(ledger.id, releasesQuery.refetch);

  return (
    <article className={cn(
      "flex min-h-44 flex-col rounded-lg border bg-card shadow-sm transition-colors",
      active ? "border-border-accent" : "border-border hover:border-ring/60",
    )}>
      <button type="button" onClick={() => onSelect(ledger)} className="flex flex-1 flex-col p-5 text-left">
        <span className="flex items-start justify-between gap-3">
          <span className="text-base font-semibold text-foreground">{ledger.name}</span>
          <span className="flex gap-2">{ledger.archivedAt && <StatusBadge label="ARCHIVED" tone="neutral" />}{active && <StatusBadge label="ACTIVE" tone="success" dot />}</span>
        </span>
        <span title={ledger.description || undefined} className="mt-2 min-h-10 line-clamp-2 text-sm text-muted-foreground">
          {ledger.description || "No description provided."}
        </span>
        <span className="mt-auto flex items-center gap-2 pt-4 text-xs text-muted-foreground">
          <span>{readyQuery.data ?? "—"} ready</span>
          <span aria-hidden="true">·</span>
          <span>{releasesQuery.data ?? "—"} releases</span>
        </span>
      </button>
      <div className="flex items-center justify-between gap-3 border-t border-border/60 px-5 py-3">
        <HashChip value={ledger.id} />
        <Button variant="ghost" size="sm" onClick={() => onLifecycle(ledger)}>{ledger.archivedAt ? "Unarchive" : "Archive"}</Button>
      </div>
    </article>
  );
}

function LedgerGridSkeleton() {
  return (
    <div className="gy-ledger-grid" aria-label="Loading ledgers">
      {Array.from({ length: 3 }, (_, index) => (
        <div key={index} className="min-h-44 rounded-lg border border-border bg-card p-5 shadow-sm" aria-busy="true">
          <Skeleton count={3} />
        </div>
      ))}
    </div>
  );
}

function LedgerList({ query, activeId, onSelect, onLifecycle, onCreate }: { query: PaginatedQueryResult<Ledger>; activeId: string; onSelect: (ledger: Ledger) => void; onLifecycle: (ledger: Ledger) => void; onCreate: () => void }) {
  if (query.loading && query.data === undefined) return <LedgerGridSkeleton />;
  if (query.error) return <ErrorState message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><LedgerGridSkeleton /></div>;
  if (query.data.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card shadow-sm">
        <EmptyState
          title="No ledgers yet"
          description="A ledger is a governed namespace with its own inbox, proposals, and release history. Create one to begin governing context."
          action={<Button onClick={onCreate}>Create your first ledger</Button>}
        />
      </div>
    );
  }
  return (
    <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>
      <div className="gy-ledger-grid">
        {query.data.map((ledger) => <LedgerCard key={ledger.id} ledger={ledger} active={ledger.id === activeId} onSelect={onSelect} onLifecycle={onLifecycle} />)}
      </div>
      {query.nextCursor && <div className="mt-5 flex justify-center"><Button variant="secondary" loading={query.loadingMore} disabled={query.loadingMore || query.refetching} onClick={query.loadMore}>Load more</Button></div>}
      {query.loadMoreError && <div className="mt-4"><ErrorState title="Unable to load more Ledgers" message={query.loadMoreError.message} onRetry={query.loadMore} retryDisabled={query.loadingMore} /></div>}
    </div>
  );
}

export function LedgersPage() {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [includeArchived, setIncludeArchived] = useState(false);
  const [lifecycleTarget, setLifecycleTarget] = useState<Ledger | null>(null);
  const confirmationTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const newLedgerButtonRef = useRef<HTMLButtonElement>(null);
  const nameInputRef = useRef<HTMLInputElement>(null);
  const restoreHeaderFocusRef = useRef(false);
  const { ledgerId, refreshLedgers, setLedgerId } = useAppState();
  const ledgerQuery = usePaginatedQuery("ledgers", (cursor, signal) => api.ledgers({ cursor, includeArchived: includeArchived || undefined }, { signal }), [includeArchived]);
  const createMutation = useMutation(async ({ ledgerName, ledgerDescription }: { ledgerName: string; ledgerDescription: string }) => {
    const ledger = await api.createLedger({ name: ledgerName, description: ledgerDescription || undefined });
    setLedgerId(ledger.id);
    setName("");
    setDescription("");
    setSubmitted(false);
    restoreHeaderFocusRef.current = true;
    setDrawerOpen(false);
    ledgerQuery.refetch();
    void refreshLedgers();
    showConfirmation(ledger);
    return ledger;
  });
  const lifecycleMutation = useMutation(async (ledger: Ledger) => {
    if (ledger.archivedAt) {
      await api.unarchiveLedger(ledger.id);
    } else {
      await api.archiveLedger(ledger.id);
      if (ledger.id === ledgerId) setLedgerId("");
    }
    setLifecycleTarget(null);
    ledgerQuery.refetch();
    void refreshLedgers();
    setConfirmation(`${ledger.name} ${ledger.archivedAt ? "unarchived" : "archived"}.`);
  });

  const showConfirmation = (ledger: Ledger) => {
    if (confirmationTimerRef.current) clearTimeout(confirmationTimerRef.current);
    setConfirmation(`Now governing ${ledger.name}`);
    confirmationTimerRef.current = setTimeout(
      () => setConfirmation(""),
      confirmationDurationSeconds * millisecondsPerSecond,
    );
  };

  useEffect(() => () => {
    if (confirmationTimerRef.current) clearTimeout(confirmationTimerRef.current);
  }, []);

  useEffect(() => {
    if (drawerOpen) nameInputRef.current?.focus();
    if (!drawerOpen && restoreHeaderFocusRef.current) {
      restoreHeaderFocusRef.current = false;
      newLedgerButtonRef.current?.focus();
    }
  }, [drawerOpen]);

  const openDrawer = () => {
    setName("");
    setDescription("");
    setSubmitted(false);
    createMutation.reset();
    setDrawerOpen(true);
  };

  const closeDrawer = () => {
    if (drawerOpen) restoreHeaderFocusRef.current = true;
    setDrawerOpen(false);
  };

  const updateName = (value: string) => {
    setName(value);
    setSubmitted(false);
    createMutation.reset();
  };

  const updateDescription = (value: string) => {
    setDescription(value);
    setSubmitted(false);
    createMutation.reset();
  };

  const formErrors = validateLedgerForm(name, description);
  const duplicateNameError = createMutation.error instanceof ApiError && createMutation.error.code === "CONFLICT"
    ? createMutation.error.message
    : undefined;
  const mutationError = createMutation.error && !duplicateNameError ? createMutation.error : undefined;

  const create = (event: FormEvent) => {
    event.preventDefault();
    setSubmitted(true);
    if (formErrors.name || formErrors.description) return;
    void createMutation.run({ ledgerName: name.trim(), ledgerDescription: description.trim() });
  };

  const selectLedger = (ledger: Ledger) => {
    setLedgerId(ledger.id);
    showConfirmation(ledger);
  };

  return (
    <>
      <PageHeader
        eyebrow="LEDGERS"
        title="Ledgers"
        description="A ledger is a governed namespace with its own inbox, proposals, and release history."
        actions={<Button ref={newLedgerButtonRef} size="sm" iconLeft={<Plus className="size-4" aria-hidden="true" />} onClick={openDrawer}>New ledger</Button>}
      />
      {confirmation && <p role="status" className="mb-4 text-sm font-medium text-success">{confirmation}</p>}
      <label className="mb-4 flex w-fit items-center gap-2 text-sm font-medium"><input type="checkbox" checked={includeArchived} onChange={(event) => setIncludeArchived(event.target.checked)} />Archived</label>
      <LedgerList query={ledgerQuery} activeId={ledgerId} onSelect={selectLedger} onLifecycle={(ledger) => { lifecycleMutation.reset(); setLifecycleTarget(ledger); }} onCreate={openDrawer} />
      <Drawer
        open={drawerOpen}
        onClose={closeDrawer}
        title="Create ledger"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={closeDrawer}>Cancel</Button>
            <Button
              type="submit"
              form="create-ledger-form"
              loading={createMutation.pending}
              disabled={createMutation.blocked}
              title={createMutation.disabledReason}
            >
              Create ledger
            </Button>
          </div>
        }
      >
        <form id="create-ledger-form" onSubmit={create}>
          <p className="mb-5 text-sm text-muted-foreground">Create a governed namespace for related context changes and releases.</p>
          <FieldGroup>
            <Field label="Name" hint={`Required, ${ledgerNameMaxLength} characters maximum.`} error={duplicateNameError ?? (submitted ? formErrors.name : undefined)}>
              <Input ref={nameInputRef} value={name} onChange={(event) => updateName(event.target.value)} placeholder="product-docs" maxLength={ledgerNameMaxLength} required />
            </Field>
            <Field label="Description" hint={`Optional, ${ledgerDescriptionMaxLength} characters maximum.`} error={submitted ? formErrors.description : undefined}>
              <Textarea value={description} onChange={(event) => updateDescription(event.target.value)} placeholder="What context does this ledger govern?" maxLength={ledgerDescriptionMaxLength} />
            </Field>
          </FieldGroup>
          {mutationError && <div className="mt-5"><ErrorState title="Unable to create ledger" message={mutationError.message} onRetry={() => void createMutation.run({ ledgerName: name.trim(), ledgerDescription: description.trim() })} retryDisabled={createMutation.blocked} retryTitle={createMutation.disabledReason} /></div>}
        </form>
      </Drawer>
      <ConfirmDialog
        open={lifecycleTarget !== null}
        onClose={() => setLifecycleTarget(null)}
        title={lifecycleTarget?.archivedAt ? "Unarchive Ledger?" : "Archive Ledger?"}
        consequence={<div className="grid gap-3"><p>{lifecycleTarget?.archivedAt ? "The Ledger will return to the working set and accept new work." : "The Ledger will leave the working set. Its complete history remains readable."}</p>{lifecycleMutation.error && <p role="alert" className="text-danger">{lifecycleMutation.error.message}</p>}</div>}
        affectedCount={1}
        confirmLabel={lifecycleTarget?.archivedAt ? "Unarchive Ledger" : "Archive Ledger"}
        confirmLoading={lifecycleMutation.pending}
        confirmDisabled={lifecycleMutation.blocked}
        confirmTitle={lifecycleMutation.disabledReason}
        onConfirm={() => { if (lifecycleTarget) void lifecycleMutation.run(lifecycleTarget); }}
      />
    </>
  );
}

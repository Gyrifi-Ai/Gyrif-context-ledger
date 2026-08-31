import { useEffect, useState, type KeyboardEvent } from "react";
import { Plus } from "lucide-react";
import { api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { usePaginatedQuery, type PaginatedQueryResult } from "../../app/use-paginated-query";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { Skeleton } from "../../ui/feedback/skeleton";
import { Button } from "../../ui/primitives/button";
import { PageHeader } from "../../ui/layout/page-header";
import { Panel } from "../../ui/layout/panel";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { formatAge } from "../shared/time";
import { proposalTone } from "../shared/status";
import { CreateProposalDrawer } from "./create-proposal-drawer";
import { ProposalDetail } from "./proposal-detail";

type ProposalStatusFilter = Proposal["status"] | "ALL";
const proposalStatuses: ProposalStatusFilter[] = ["ALL", "DRAFT", "REVIEWED", "APPROVED", "RELEASED", "BLOCKED", "CANCELLED"];

function ProposalList({ query, selectedId, onSelect, onCreate }: { query: PaginatedQueryResult<Proposal>; selectedId?: string; onSelect: (proposal: Proposal) => void; onCreate: () => void }) {
  if (query.loading && query.data === undefined) return <Panel padding="none"><div className="grid gap-3 p-4" aria-label="Loading Proposals">{Array.from({ length: 5 }, (_, index) => <Skeleton key={index} height="4rem" />)}</div></Panel>;
  if (query.error) return <ErrorState title="Unable to load Proposals" message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><Panel><Skeleton height="18rem" /></Panel></div>;
  if (query.data.length === 0) return <Panel><EmptyState title="No Proposals" description="Select READY Changes from the inbox or start a new ordered Proposal here." action={<Button onClick={onCreate}>New proposal</Button>} /></Panel>;

  const keyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    const nextIndex = Math.max(0, Math.min(query.data!.length - 1, index + (event.key === "ArrowDown" ? 1 : -1)));
    onSelect(query.data![nextIndex]);
    event.currentTarget.parentElement?.parentElement?.querySelectorAll<HTMLButtonElement>("button[data-proposal]")[nextIndex]?.focus();
  };

  return (
    <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>
      <Panel padding="none" title="Review queue" description="Select a Context PR to inspect its evidence and gates.">
        <ul className="divide-y divide-border">
          {query.data.map((proposal, index) => (
            <li key={proposal.id}>
              <button
                type="button"
                data-proposal
                onClick={() => onSelect(proposal)}
                onKeyDown={(event) => keyDown(event, index)}
                className={`grid w-full gap-2 border-l-2 px-4 py-4 text-left hover:bg-muted/60 ${selectedId === proposal.id ? "border-l-primary bg-primary/5" : "border-l-transparent"}`}
                aria-current={selectedId === proposal.id ? "true" : undefined}
              >
                <span className="flex items-center justify-between gap-3"><span className="truncate font-medium">{proposal.title}</span><StatusBadge label={proposal.status} tone={proposalTone(proposal.status)} /></span>
                <span className="flex items-center justify-between gap-3 text-xs text-muted-foreground"><span>{proposal.changeIds.length} changes</span><time dateTime={proposal.createdAt} title={proposal.createdAt}>{formatAge(proposal.createdAt)}</time></span>
              </button>
            </li>
          ))}
        </ul>
        {query.nextCursor && <div className="border-t border-border p-3 text-center"><Button variant="secondary" size="sm" loading={query.loadingMore} disabled={query.loadingMore || query.refetching} onClick={query.loadMore}>Load more</Button></div>}
        {query.loadMoreError && <div className="border-t border-border p-3"><ErrorState title="Unable to load more Proposals" message={query.loadMoreError.message} onRetry={query.loadMore} retryDisabled={query.loadingMore} /></div>}
      </Panel>
    </div>
  );
}

export function ProposalsPage({ proposalId }: { proposalId?: string }) {
  const { ledgerId, openLedgerSwitcher } = useAppState();
  const [createOpen, setCreateOpen] = useState(false);
  const [status, setStatus] = useState<ProposalStatusFilter>("ALL");
  const proposalsQuery = usePaginatedQuery("proposals", (cursor, signal) => ledgerId ? api.proposals(ledgerId, { cursor, status: status === "ALL" ? undefined : status }, { signal }) : Promise.resolve({ items: [] }), [ledgerId, status]);
  const readyChangesQuery = usePaginatedQuery("proposal-ready-changes", (cursor, signal) => ledgerId ? api.changes(ledgerId, { cursor, status: "READY" }, { signal }) : Promise.resolve({ items: [] }), [ledgerId]);
  useLedgerEvents(ledgerId, () => { proposalsQuery.refetch(); readyChangesQuery.refetch(); });

  useEffect(() => {
    if (!proposalId && proposalsQuery.data?.[0]) window.location.hash = `proposals/${proposalsQuery.data[0].id}`;
  }, [proposalId, proposalsQuery.data]);

  const select = (proposal: Proposal) => { window.location.hash = `proposals/${proposal.id}`; };
  const created = (proposal: Proposal) => {
    setCreateOpen(false);
    proposalsQuery.refetch();
    readyChangesQuery.refetch();
    select(proposal);
  };

  if (!ledgerId) return <><PageHeader eyebrow="CONTEXT PRs" title="Proposals" description="Review batched changes, attach evidence, approve, and release." /><Panel><EmptyState title="Select a ledger" description="Choose a governed namespace before reviewing Proposals." action={<Button onClick={openLedgerSwitcher}>Select ledger</Button>} /></Panel></>;

  return (
    <>
      <PageHeader eyebrow="CONTEXT PRs" title="Proposals" description="Review batched changes, attach evidence, approve, and release." actions={<Button size="sm" iconLeft={<Plus className="size-4" aria-hidden="true" />} onClick={() => setCreateOpen(true)}>New proposal</Button>} />
      <div className="mb-4 flex justify-end"><label className="grid gap-1 text-xs font-medium text-muted-foreground">Status<select aria-label="Filter Proposals by status" value={status} onChange={(event) => setStatus(event.target.value as ProposalStatusFilter)} className="h-9 rounded-sm border border-input bg-muted px-3 text-sm text-foreground">{proposalStatuses.map((value) => <option key={value} value={value}>{value === "ALL" ? "All statuses" : value}</option>)}</select></label></div>
      <div className="gy-proposal-workspace">
        <ProposalList query={proposalsQuery} selectedId={proposalId} onSelect={select} onCreate={() => setCreateOpen(true)} />
        {proposalId ? <ProposalDetail ledgerId={ledgerId} proposalId={proposalId} onUpdated={() => { proposalsQuery.refetch(); readyChangesQuery.refetch(); }} /> : <Panel><EmptyState title="Select a Proposal" description="Choose a Context PR from the review queue to inspect its evidence, approval, and release gates." /></Panel>}
      </div>
      <CreateProposalDrawer open={createOpen} ledgerId={ledgerId} changes={readyChangesQuery.data ?? []} hasMoreChanges={Boolean(readyChangesQuery.nextCursor)} loadingMoreChanges={readyChangesQuery.loadingMore} onLoadMoreChanges={readyChangesQuery.loadMore} onClose={() => setCreateOpen(false)} onCreated={created} onConflict={readyChangesQuery.refetch} />
    </>
  );
}

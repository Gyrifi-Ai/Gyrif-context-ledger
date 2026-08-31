import { useEffect, useState, type KeyboardEvent } from "react";
import { Plus } from "lucide-react";
import { api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useQuery, type QueryResult } from "../../app/use-async";
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

type ProposalWorkspace = { proposals: Proposal[]; readyChanges: Change[] };

function ProposalList({ query, selectedId, onSelect, onCreate }: { query: QueryResult<ProposalWorkspace>; selectedId?: string; onSelect: (proposal: Proposal) => void; onCreate: () => void }) {
  if (query.loading && query.data === undefined) return <Panel padding="none"><div className="grid gap-3 p-4" aria-label="Loading Proposals">{Array.from({ length: 5 }, (_, index) => <Skeleton key={index} height="4rem" />)}</div></Panel>;
  if (query.error) return <ErrorState title="Unable to load Proposals" message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><Panel><Skeleton height="18rem" /></Panel></div>;
  if (query.data.proposals.length === 0) return <Panel><EmptyState title="No Proposals" description="Select READY Changes from the inbox or start a new ordered Proposal here." action={<Button onClick={onCreate}>New proposal</Button>} /></Panel>;

  const keyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    const nextIndex = Math.max(0, Math.min(query.data!.proposals.length - 1, index + (event.key === "ArrowDown" ? 1 : -1)));
    onSelect(query.data!.proposals[nextIndex]);
    event.currentTarget.parentElement?.parentElement?.querySelectorAll<HTMLButtonElement>("button[data-proposal]")[nextIndex]?.focus();
  };

  return (
    <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>
      <Panel padding="none" title="Review queue" description="Select a Context PR to inspect its evidence and gates.">
        <ul className="divide-y divide-border">
          {query.data.proposals.map((proposal, index) => (
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
      </Panel>
    </div>
  );
}

export function ProposalsPage({ proposalId }: { proposalId?: string }) {
  const { ledgerId, openLedgerSwitcher } = useAppState();
  const [createOpen, setCreateOpen] = useState(false);
  const workspaceQuery = useQuery("proposals", async (signal) => {
    if (!ledgerId) return { proposals: [], readyChanges: [] };
    const [proposals, changes] = await Promise.all([api.proposals(ledgerId, { signal }), api.changes(ledgerId, { signal })]);
    return { proposals: proposals.items ?? [], readyChanges: (changes.items ?? []).filter((change) => change.status === "READY") };
  }, [ledgerId]);
  useLedgerEvents(ledgerId, workspaceQuery.refetch);

  useEffect(() => {
    if (!proposalId && workspaceQuery.data?.proposals[0]) window.location.hash = `proposals/${workspaceQuery.data.proposals[0].id}`;
  }, [proposalId, workspaceQuery.data]);

  const select = (proposal: Proposal) => { window.location.hash = `proposals/${proposal.id}`; };
  const created = (proposal: Proposal) => {
    setCreateOpen(false);
    workspaceQuery.refetch();
    select(proposal);
  };

  if (!ledgerId) return <><PageHeader eyebrow="CONTEXT PRs" title="Proposals" description="Review batched changes, attach evidence, approve, and release." /><Panel><EmptyState title="Select a ledger" description="Choose a governed namespace before reviewing Proposals." action={<Button onClick={openLedgerSwitcher}>Select ledger</Button>} /></Panel></>;

  return (
    <>
      <PageHeader eyebrow="CONTEXT PRs" title="Proposals" description="Review batched changes, attach evidence, approve, and release." actions={<Button size="sm" iconLeft={<Plus className="size-4" aria-hidden="true" />} onClick={() => setCreateOpen(true)}>New proposal</Button>} />
      <div className="gy-proposal-workspace">
        <ProposalList query={workspaceQuery} selectedId={proposalId} onSelect={select} onCreate={() => setCreateOpen(true)} />
        {proposalId ? <ProposalDetail ledgerId={ledgerId} proposalId={proposalId} onUpdated={workspaceQuery.refetch} /> : <Panel><EmptyState title="Select a Proposal" description="Choose a Context PR from the review queue to inspect its evidence, approval, and release gates." /></Panel>}
      </div>
      <CreateProposalDrawer open={createOpen} ledgerId={ledgerId} changes={workspaceQuery.data?.readyChanges ?? []} onClose={() => setCreateOpen(false)} onCreated={created} onConflict={workspaceQuery.refetch} />
    </>
  );
}

import { useEffect, useState } from "react";
import type { Approval, Change, CheckResult, ProposalDetail as ProposalDetailData } from "../../api/types";
import { api } from "../../api/client";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useMutation, useQuery } from "../../app/use-async";
import { ChangeDetailDrawer } from "../changes/change-detail-drawer";
import { changeTone, proposalTone } from "../shared/status";
import { ErrorState } from "../../ui/feedback/error-state";
import { Skeleton } from "../../ui/feedback/skeleton";
import { Button } from "../../ui/primitives/button";
import { Panel } from "../../ui/layout/panel";
import { DataTable, type Column } from "../../ui/patterns/data-table";
import { ConfirmDialog } from "../../ui/patterns/confirm-dialog";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { ApprovalPanel } from "./approval-panel";
import { EvidencePanel } from "./evidence-panel";
import { ProgressRail } from "./progress-rail";
import { progressSteps } from "./proposal-view";
import { ReleasePanel } from "./release-panel";
import { approvalGate, cancelGate } from "./gates";

const changeColumns: Column<Change>[] = [
  { key: "ordinal", header: "#", align: "end", render: (_change, index) => index + 1 },
  { key: "unit", header: "UNIT", mono: true, render: (change) => change.unit },
  { key: "action", header: "ACTION", render: (change) => change.action },
  { key: "fingerprint", header: "DESIRED FINGERPRINT", render: (change) => <span onClick={(event) => event.stopPropagation()}><HashChip value={change.desiredFingerprint} /></span> },
  { key: "status", header: "STATUS", render: (change) => <StatusBadge label={change.status} tone={changeTone(change.status)} /> },
];

type DetailWorkspace = { detail: ProposalDetailData; checks: CheckResult[]; approvals: Approval[] };

function DetailSkeleton() {
  return <Panel><div className="grid gap-4" aria-label="Loading Proposal detail"><Skeleton height="1.5rem" width="45%" /><Skeleton height="3.5rem" /><Skeleton height="8rem" /><Skeleton height="8rem" /></div></Panel>;
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <details open className="group border-b border-border last:border-b-0">
      <summary className="cursor-pointer list-none px-4 py-3 font-semibold marker:hidden">{title}</summary>
      <div className="border-t border-border bg-card p-4">{children}</div>
    </details>
  );
}

export function ProposalDetail({ ledgerId, proposalId, onUpdated }: { ledgerId: string; proposalId: string; onUpdated: () => void }) {
  const [selectedChange, setSelectedChange] = useState<Change | null>(null);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const query = useQuery("proposal-detail", async (signal) => {
    const [detail, checks, approvals] = await Promise.all([
      api.proposal(ledgerId, proposalId, { signal }),
      api.proposalChecks(ledgerId, proposalId, { signal }),
      api.proposalApprovals(ledgerId, proposalId, { signal }),
    ]);
    return { detail, checks: checks.items ?? [], approvals: approvals.items ?? [] };
  }, [ledgerId, proposalId]);
  useLedgerEvents(ledgerId, query.refetch);

  const refresh = () => {
    query.refetch();
    onUpdated();
  };
  const cancelMutation = useMutation(async (_: undefined) => {
    await api.cancelProposal(ledgerId, proposalId);
    setConfirmCancel(false);
    refresh();
  });
  useEffect(() => {
    setConfirmCancel(false);
    cancelMutation.reset();
  }, [proposalId]);

  if (query.loading && query.data === undefined) return <DetailSkeleton />;
  if (query.error) return <ErrorState title="Unable to load Proposal detail" message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><DetailSkeleton /></div>;

  const workspace: DetailWorkspace = query.data;
  const { detail, checks, approvals } = workspace;
  const displayedCheck = checks[0];
  const currentApproval = approvals.find((approval) => approval.current);
  const cancellation = cancelGate(detail.gates);

  return (
    <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>
      <Panel padding="none">
        <div className="grid gap-4 p-4">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="mb-2 flex flex-wrap items-center gap-2"><h2 className="text-lg font-semibold">{detail.proposal.title}</h2><StatusBadge label={detail.proposal.status} tone={proposalTone(detail.proposal.status)} /></div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground"><span>Proposal</span><HashChip value={detail.proposal.id} /><span>Hash</span><HashChip value={detail.proposal.hash} /></div>
            </div>
            <div className="text-right text-xs text-muted-foreground"><p className="mb-1">Base release</p>{detail.proposal.baseReleaseId ? <HashChip value={detail.proposal.baseReleaseId} /> : <span className="font-mono">initial HEAD</span>}</div>
          </div>
          <ProgressRail steps={progressSteps(detail, displayedCheck)} />
        </div>
        <Section title={`Changes (${detail.changes.length})`}>
          <DataTable columns={changeColumns} rows={detail.changes} getRowId={(change) => change.id} onRowClick={setSelectedChange} empty={<p className="text-sm text-muted-foreground">No Changes are attached.</p>} />
        </Section>
        <Section title="Evidence"><EvidencePanel ledgerId={ledgerId} proposal={detail.proposal} check={displayedCheck} onRefresh={refresh} /></Section>
        <Section title="Approval"><ApprovalPanel ledgerId={ledgerId} proposal={detail.proposal} approval={currentApproval} gate={approvalGate(detail.gates)} onRefresh={refresh} /></Section>
        <Section title="Release"><ReleasePanel ledgerId={ledgerId} detail={detail} onRefresh={refresh} /></Section>
        <div className="flex flex-wrap items-center gap-3 p-4">
          <Button variant="secondary" loading={cancelMutation.pending} disabled={!cancellation.enabled || cancelMutation.blocked} title={!cancellation.enabled ? cancellation.reason : cancelMutation.disabledReason} onClick={() => setConfirmCancel(true)}>Cancel Proposal</Button>
          {!cancellation.enabled && cancellation.reason && <span className="text-xs text-muted-foreground">{cancellation.reason}</span>}
        </div>
      </Panel>
      <ChangeDetailDrawer change={selectedChange} onClose={() => setSelectedChange(null)} onWithdraw={() => undefined} withdrawBlocked withdrawDisabledReason="Cancel the Proposal first." />
      <ConfirmDialog open={confirmCancel} onClose={() => setConfirmCancel(false)} title="Cancel Proposal?" consequence={<p>The Changes will return to the inbox and can be selected into a new Proposal. Existing evidence and approvals remain in the audit trail.</p>} affectedCount={detail.changes.length} confirmLabel="Cancel Proposal" confirmLoading={cancelMutation.pending} confirmDisabled={!cancellation.enabled || cancelMutation.blocked} confirmTitle={!cancellation.enabled ? cancellation.reason : cancelMutation.disabledReason} onConfirm={() => void cancelMutation.run(undefined)} />
      {cancelMutation.error && <ErrorState title="Unable to cancel Proposal" message={cancelMutation.error.message} onRetry={() => setConfirmCancel(true)} retryDisabled={cancelMutation.blocked || !cancellation.enabled} retryTitle={!cancellation.enabled ? cancellation.reason : cancelMutation.disabledReason} />}
    </div>
  );
}

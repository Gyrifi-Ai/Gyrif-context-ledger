import { useEffect, useState } from "react";
import { CheckCircle2 } from "lucide-react";
import { api } from "../../api/client";
import type { Proposal, Release, ReleaseIntent } from "../../api/types";
import { useAppState } from "../../app/providers";
import { useLedgerEvents } from "../../app/use-ledger-events";
import { useQuery } from "../../app/use-async";
import { Button } from "../../ui/primitives/button";
import { EmptyState } from "../../ui/feedback/empty-state";
import { ErrorState } from "../../ui/feedback/error-state";
import { Skeleton } from "../../ui/feedback/skeleton";
import { PageHeader } from "../../ui/layout/page-header";
import { Panel } from "../../ui/layout/panel";
import { PlanDrawer } from "./plan-drawer";
import { RecoveryBanner } from "./recovery-banner";
import { ReleaseTimeline } from "./release-timeline";
import { RollbackDialog } from "./rollback-dialog";

type ReleasesWorkspace = { releases: Release[]; proposals: Proposal[]; intents: ReleaseIntent[] };
type SelectedPlan = { release: Release; intent: ReleaseIntent };
type RollbackTarget = { release: Release; affectedUnitCount: number };

function ReleasesLoading() {
  return <Panel title="Release timeline" description="Loading immutable history and recovery state."><div className="grid gap-6" aria-label="Loading Releases">{Array.from({ length: 4 }, (_, index) => <div key={index} className="grid grid-cols-[0.75rem_1fr] gap-4"><Skeleton width="0.75rem" height="0.75rem" radius="999px" /><Skeleton height="5.5rem" /></div>)}</div></Panel>;
}

export function RollbackSuccess({ proposal }: { proposal: Proposal }) {
  return <div role="status" className="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-md border border-success/40 bg-success/10 p-4"><p className="flex items-center gap-2 text-sm font-medium text-success"><CheckCircle2 className="size-4" aria-hidden="true" />Rollback proposal created: {proposal.title}</p><a href={`#proposals/${proposal.id}`}><Button size="sm">Review proposal</Button></a></div>;
}

export function ReleasesPage() {
  const { ledgerId, openLedgerSwitcher } = useAppState();
  const [selectedPlan, setSelectedPlan] = useState<SelectedPlan>();
  const [rollbackTarget, setRollbackTarget] = useState<RollbackTarget>();
  const [createdProposal, setCreatedProposal] = useState<Proposal>();
  const [recoveryNotice, setRecoveryNotice] = useState<string>();
  const workspace = useQuery("releases-workspace", async (signal): Promise<ReleasesWorkspace> => {
    if (!ledgerId) return { releases: [], proposals: [], intents: [] };
    const [releases, proposals, intents] = await Promise.all([api.releases(ledgerId, { signal }), api.proposals(ledgerId, { signal }), api.releaseIntents(ledgerId, undefined, { signal })]);
    return { releases: releases.items ?? [], proposals: proposals.items ?? [], intents: intents.items ?? [] };
  }, [ledgerId]);
  useLedgerEvents(ledgerId, workspace.refetch);
  useEffect(() => {
    if (!recoveryNotice) return;
    const timer = window.setTimeout(() => setRecoveryNotice(undefined), 3_000);
    return () => window.clearTimeout(timer);
  }, [recoveryNotice]);

  if (!ledgerId) return <><PageHeader eyebrow="IMMUTABLE HISTORY" title="Releases" description="Every release was applied to the target and verified before it was recorded." /><Panel><EmptyState title="Select a ledger" description="Release history and recovery are scoped to a governed namespace." action={<Button onClick={openLedgerSwitcher}>Select ledger</Button>} /></Panel></>;

  let content;
  if (workspace.loading && workspace.data === undefined) content = <ReleasesLoading />;
  else if (workspace.error) content = <ErrorState title="Unable to load Releases" message={workspace.error.message} onRetry={workspace.refetch} />;
  else if (workspace.data === undefined) content = <div className={workspace.unavailable ? "gy-is-refetching" : undefined}><ReleasesLoading /></div>;
  else {
    const recoveryIntents = workspace.data.intents.filter((intent) => intent.status === "RECOVERY_REQUIRED");
    content = (
      <div className={workspace.refetching || workspace.unavailable ? "gy-is-refetching" : undefined}>
        <RecoveryBanner ledgerId={ledgerId} intents={recoveryIntents} onUpdated={workspace.refetch} onResolved={setRecoveryNotice} />
        {recoveryNotice && <div role="status" className="mb-5 flex items-center gap-2 rounded-md border border-success/40 bg-success/10 p-4 text-sm font-medium text-success"><CheckCircle2 className="size-4" aria-hidden="true" />{recoveryNotice}</div>}
        {createdProposal && <RollbackSuccess proposal={createdProposal} />}
        {workspace.data.releases.length === 0 ? <Panel><EmptyState title="No Releases yet" description="Approved Proposals become immutable Releases here after they are applied and verified." action={<a className="text-sm font-medium text-primary underline" href="#proposals">Review Proposals</a>} /></Panel> : <Panel title="Release timeline" description={`${workspace.data.releases.length} immutable ${workspace.data.releases.length === 1 ? "release" : "releases"}, newest first.`}><ReleaseTimeline releases={workspace.data.releases} proposals={workspace.data.proposals} intents={workspace.data.intents} onViewPlan={(release, intent) => setSelectedPlan({ release, intent })} onRollback={(release, affectedUnitCount) => { setCreatedProposal(undefined); setRollbackTarget({ release, affectedUnitCount }); }} /></Panel>}
      </div>
    );
  }

  return (
    <>
      <PageHeader eyebrow="IMMUTABLE HISTORY" title="Releases" description="Every release was applied to the target and verified before it was recorded." />
      {content}
      <PlanDrawer open={Boolean(selectedPlan)} onClose={() => setSelectedPlan(undefined)} release={selectedPlan?.release} operations={selectedPlan?.intent.plan.operations} />
      <RollbackDialog open={Boolean(rollbackTarget)} onClose={() => setRollbackTarget(undefined)} ledgerId={ledgerId} release={rollbackTarget?.release} affectedUnitCount={rollbackTarget?.affectedUnitCount ?? 0} onCreated={(proposal) => { setCreatedProposal(proposal); workspace.refetch(); }} />
    </>
  );
}

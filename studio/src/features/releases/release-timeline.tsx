import type { Proposal, Release, ReleaseIntent } from "../../api/types";
import { Button } from "../../ui/primitives/button";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { Timeline } from "../../ui/patterns/timeline";
import { formatAge } from "../shared/time";

export function intentForRelease(release: Release, intents: ReleaseIntent[]): ReleaseIntent | undefined {
  return intents.find((intent) => intent.status === "FINALIZED" && intent.proposalId === release.proposalId);
}

export function rollbackAffectedUnitCount(releases: Release[], targetReleaseId: string, intents: ReleaseIntent[]): number | undefined {
  const targetIndex = releases.findIndex((release) => release.id === targetReleaseId);
  if (targetIndex < 1) return targetIndex === 0 ? 0 : undefined;
  const units = new Set<string>();
  for (const release of releases.slice(0, targetIndex)) {
    const intent = intentForRelease(release, intents);
    if (!intent) return undefined;
    intent.plan.operations.forEach((operation) => units.add(operation.unit));
  }
  return units.size;
}

export function ReleaseTimeline({ releases, proposals, intents, onViewPlan, onRollback }: { releases: Release[]; proposals: Proposal[]; intents: ReleaseIntent[]; onViewPlan: (release: Release, intent: ReleaseIntent) => void; onRollback: (release: Release, affectedUnitCount: number) => void }) {
  const proposalTitles = new Map(proposals.map((proposal) => [proposal.id, proposal.title]));
  return (
    <Timeline items={releases.map((release, index) => {
      const intent = intentForRelease(release, intents);
      const affectedUnitCount = rollbackAffectedUnitCount(releases, release.id, intents);
      return {
        id: release.id,
        current: index === 0,
        node: <div className="mb-2 flex items-center justify-between gap-3"><span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{index === 0 ? "Current release" : "History"}</span>{index === 0 && <StatusBadge label="HEAD" tone="success" dot />}</div>,
        title: <HashChip value={release.id} label="Release" />,
        meta: <div className="flex flex-wrap items-center gap-x-2 gap-y-1"><span className="font-medium text-foreground">{proposalTitles.get(release.proposalId) ?? `Proposal ${release.proposalId}`}</span><span>·</span><span>{intent?.plan.operations.length ?? "—"} units</span><span>·</span><HashChip value={release.hash} label="Hash" /><span>·</span><time dateTime={release.createdAt} title={release.createdAt}>{formatAge(release.createdAt)}</time></div>,
        body: <div className="flex flex-wrap gap-2"><Button variant="secondary" size="sm" disabled={!intent} title={intent ? undefined : "Release plan is unavailable."} onClick={() => intent && onViewPlan(release, intent)}>View plan</Button>{index > 0 && <Button variant="danger" size="sm" disabled={affectedUnitCount === undefined} title={affectedUnitCount === undefined ? "Rollback plan is unavailable." : undefined} onClick={() => affectedUnitCount !== undefined && onRollback(release, affectedUnitCount)}>Roll back to here</Button>}</div>,
      };
    })} />
  );
}
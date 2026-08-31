import type { CheckResult, EvaluationResponse, Finding, ProposalDetail } from "../../api/types";

export type ProgressState = "complete" | "current" | "pending";
export type ProgressStep = { label: "Changes" | "Evidence" | "Approval" | "Release"; state: ProgressState };
export type EvidenceView = {
  findings: Finding[];
  model?: string;
  previewFidelity?: string;
};

function record(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : undefined;
}

function findings(value: unknown): Finding[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    const candidate = record(item);
    if (!candidate || typeof candidate.severity !== "string" || typeof candidate.message !== "string") return [];
    return [{ severity: candidate.severity, message: candidate.message, unit: typeof candidate.unit === "string" ? candidate.unit : undefined }];
  });
}

export function evidenceView(check: CheckResult | undefined, latestResponse?: EvaluationResponse): EvidenceView {
  const evidence = record(check?.evidence);
  const preview = record(evidence?.preview);
  return {
    findings: findings(evidence?.findings ?? latestResponse?.findings),
    model: typeof evidence?.model === "string" ? evidence.model : undefined,
    previewFidelity: typeof evidence?.previewFidelity === "string"
      ? evidence.previewFidelity
      : typeof preview?.fidelity === "string"
        ? preview.fidelity
        : latestResponse?.previewFidelity,
  };
}

export function progressSteps(detail: ProposalDetail, displayedCheck?: CheckResult): ProgressStep[] {
  const staleDisplayedEvidence = Boolean(displayedCheck && !displayedCheck.current);
  const evidenceComplete = detail.gates.hasCurrentPassingCheck && !staleDisplayedEvidence;
  const approvalComplete = evidenceComplete && detail.gates.hasCurrentApproval;
  const releaseComplete = detail.proposal.status === "RELEASED";
  const complete = [true, evidenceComplete, approvalComplete, releaseComplete];
  const firstIncomplete = complete.findIndex((value) => !value);
  const labels: ProgressStep["label"][] = ["Changes", "Evidence", "Approval", "Release"];
  return labels.map((label, index) => ({
    label,
    state: complete[index] ? "complete" : index === firstIncomplete ? "current" : "pending",
  }));
}

export function moveOrdered(ids: string[], index: number, direction: -1 | 1): string[] {
  const destination = index + direction;
  if (destination < 0 || destination >= ids.length) return ids;
  const next = [...ids];
  [next[index], next[destination]] = [next[destination], next[index]];
  return next;
}

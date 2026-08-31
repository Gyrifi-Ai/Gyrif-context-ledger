export type Ledger = {
  id: string;
  name: string;
  description: string;
  createdAt: string;
};

export type Change = {
  id: string;
  ledgerId: string;
  sequence: number;
  unit: string;
  action: "PUT" | "DELETE";
  desired: unknown;
  baseFingerprint: string;
  desiredFingerprint: string;
  status: "ACCEPTED" | "READY" | "INVALID" | "RELEASED";
  createdAt: string;
};

export type Proposal = {
  id: string;
  ledgerId: string;
  title: string;
  baseReleaseId?: string;
  hash: string;
  status: "DRAFT" | "REVIEWED" | "APPROVED" | "RELEASED" | "BLOCKED";
  changeIds: string[];
  createdAt: string;
};

export type ActionGate = {
  enabled: boolean;
  reason: string;
};

export type ProposalGates = {
  hasCurrentPassingCheck: boolean;
  hasCurrentApproval: boolean;
  baseMatchesHead: boolean;
  releasable: boolean;
  reason: string;
  approvalAction: ActionGate;
  releaseAction: ActionGate;
};

export type ProposalDetail = {
  proposal: Proposal;
  changes: Change[];
  currentHeadReleaseId: string;
  gates: ProposalGates;
};

export type CheckResult = {
  id: string;
  proposalHash: string;
  kind: string;
  passed: boolean;
  summary: string;
  evidence?: unknown;
  evidenceUnavailable?: boolean;
  createdAt: string;
  current: boolean;
};

export type Finding = {
  severity: string;
  unit?: string;
  message: string;
};

export type EvaluationResponse = {
  passed: boolean;
  summary: string;
  previewFidelity: string;
  findings?: Finding[];
};

export type Approval = {
  id: string;
  proposalHash: string;
  actor: string;
  createdAt: string;
  current: boolean;
};

export type Release = {
  id: string;
  ledgerId: string;
  proposalId: string;
  parentId?: string;
  hash: string;
  createdAt: string;
};

export type SystemStatus = {
  status: string;
  version: string;
  inference: string;
};

export type EventKind =
  | "change.accepted"
  | "proposal.created"
  | "proposal.evaluated"
  | "proposal.approved"
  | "release.started"
  | "release.completed"
  | "release.failed"
  | "intent.recovery_required";

export type DomainEvent = {
  kind: EventKind;
  ledgerId: string;
  subjectId: string;
  at: string;
};

export type APIError = { error: { code: string; message: string } };
export type ReleaseIntentStatus = "READY" | "APPLYING" | "VERIFYING" | "FINALIZED" | "RECOVERY_REQUIRED";

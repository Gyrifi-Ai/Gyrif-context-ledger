export type Ledger = {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  archivedAt?: string;
};

export type ListOptions = {
  limit?: number;
  cursor?: string;
  status?: string;
  action?: string;
  includeArchived?: boolean;
};

export type ListPage<T> = {
  items: T[];
  nextCursor?: string;
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
  status: "ACCEPTED" | "READY" | "INVALID" | "RELEASED" | "WITHDRAWN";
  invalidReason?: string;
  noop?: boolean;
  stalled?: boolean;
  createdAt: string;
};

export type Proposal = {
  id: string;
  ledgerId: string;
  title: string;
  baseReleaseId?: string;
  hash: string;
  status: "DRAFT" | "REVIEWED" | "APPROVED" | "RELEASED" | "BLOCKED" | "CANCELLED";
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
  cancelAction: ActionGate;
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
  commit: string;
  buildDate: string;
  inference: string;
  health: {
    database: "ok" | "unreachable";
    target: "ok" | "unreachable" | "unknown";
    inference: "ok" | "disabled" | "unhealthy";
    inferenceState: "ready" | "starting" | "restarting" | "failed" | "stopped" | "disabled";
    unresolvedIntents: number;
  };
};

export type EventKind =
  | "change.accepted"
  | "change.withdrawn"
  | "ledger.archived"
  | "ledger.unarchived"
  | "proposal.created"
  | "proposal.evaluated"
  | "proposal.approved"
  | "proposal.cancelled"
  | "release.started"
  | "release.completed"
  | "release.failed"
  | "intent.recovery_required"
  | "intent.resolved";

export type DomainEvent = {
  kind: EventKind;
  ledgerId: string;
  subjectId: string;
  at: string;
};

export type APIError = { error: { code: string; message: string } };
export type ReleaseIntentStatus = "READY" | "APPLYING" | "VERIFYING" | "FINALIZED" | "RECOVERY_REQUIRED" | "ABANDONED";

export type ReleaseIntentOperation = {
  unit: string;
  action: Change["action"];
  desired?: unknown;
  expectedFingerprint?: string;
  expectedExists: boolean;
  desiredFingerprint: string;
  targetMetric?: string;
  beforeObjectHash?: string;
  beforeExists: boolean;
  hasBeforeImage: boolean;
};

export type ReleaseIntent = {
  id: string;
  ledgerId: string;
  proposalId: string;
  proposalHash: string;
  parentId?: string;
  status: ReleaseIntentStatus;
  plan: { operations: ReleaseIntentOperation[] };
  createdAt: string;
  resolution?: string;
  resolutionNote?: string;
  resolvedAt?: string;
};

export type RetryReleaseIntentResult = {
  resolved: boolean;
  mismatches: Array<{ unit: string; expected: string; observed: string }>;
};

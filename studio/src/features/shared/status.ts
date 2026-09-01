import type { Change, Proposal, ReleaseIntentStatus } from "../../api/types";
import type { StatusTone } from "../../ui/patterns/status-badge";

function assertNever(value: never): never { throw new Error(`Unhandled value: ${String(value)}`); }
export function changeTone(status: Change["status"]): StatusTone { switch (status) { case "ACCEPTED": return "info"; case "READY": case "WITHDRAWN": return "neutral"; case "INVALID": return "danger"; case "RELEASED": return "success"; default: return assertNever(status); } }
export function proposalTone(status: Proposal["status"]): StatusTone { switch (status) { case "DRAFT": case "CANCELLED": return "neutral"; case "REVIEWED": return "review"; case "APPROVED": case "RELEASED": return "success"; case "BLOCKED": return "danger"; default: return assertNever(status); } }
export function intentTone(status: ReleaseIntentStatus): StatusTone { switch (status) { case "READY": case "APPLYING": case "VERIFYING": return "warning"; case "FINALIZED": return "success"; case "RECOVERY_REQUIRED": return "danger"; case "ABANDONED": return "neutral"; default: return assertNever(status); } }

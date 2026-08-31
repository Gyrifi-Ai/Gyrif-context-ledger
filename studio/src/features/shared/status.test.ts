import { describe, expect, it } from "vitest";
import type { Change, Proposal, ReleaseIntentStatus } from "../../api/types";
import type { StatusTone } from "../../ui/patterns/status-badge";
import { changeTone, intentTone, proposalTone } from "./status";

const changeTones: Record<Change["status"], StatusTone> = { ACCEPTED: "info", READY: "neutral", INVALID: "danger", RELEASED: "success" };
const proposalTones: Record<Proposal["status"], StatusTone> = { DRAFT: "neutral", REVIEWED: "review", APPROVED: "success", RELEASED: "success", BLOCKED: "danger" };
const intentTones: Record<ReleaseIntentStatus, StatusTone> = { READY: "warning", APPLYING: "warning", VERIFYING: "warning", FINALIZED: "success", RECOVERY_REQUIRED: "danger", ABANDONED: "neutral" };

describe("domain status tones", () => {
  it("maps every API status to its normative tone", () => {
    Object.entries(changeTones).forEach(([status, tone]) => expect(changeTone(status as Change["status"])).toBe(tone));
    Object.entries(proposalTones).forEach(([status, tone]) => expect(proposalTone(status as Proposal["status"])).toBe(tone));
    Object.entries(intentTones).forEach(([status, tone]) => expect(intentTone(status as ReleaseIntentStatus)).toBe(tone));
  });
});

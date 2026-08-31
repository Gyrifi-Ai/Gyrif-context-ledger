import { describe, expect, it } from "vitest";
import type { CheckResult, ProposalDetail } from "../../api/types";
import { evidenceView, moveOrdered, progressSteps } from "./proposal-view";
import { approvalGate, cancelGate, releaseGate } from "./gates";

function detail(overrides: Partial<ProposalDetail["gates"]> = {}, status: ProposalDetail["proposal"]["status"] = "DRAFT"): ProposalDetail {
  return {
    proposal: { id: "pr_one", ledgerId: "ldg_one", title: "Review", hash: "sha256:current", status, changeIds: ["chg_one"], createdAt: "2026-08-31T00:00:00Z" },
    changes: [],
    currentHeadReleaseId: "",
    gates: {
      hasCurrentPassingCheck: false,
      hasCurrentApproval: false,
      baseMatchesHead: true,
      releasable: false,
      reason: "A current passing evaluation is required.",
      approvalAction: { enabled: false, reason: "A current passing evaluation is required before approval." },
      releaseAction: { enabled: false, reason: "A current passing evaluation is required." },
      cancelAction: { enabled: true, reason: "" },
      ...overrides,
    },
  };
}

const currentCheck: CheckResult = { id: "chk_one", proposalHash: "sha256:current", kind: "natural-language", passed: true, summary: "Safe", evidence: { findings: [{ severity: "warning", unit: "point/1", message: "Review wording" }], model: "gemma", previewFidelity: "FAST" }, createdAt: "2026-08-31T00:01:00Z", current: true };

describe("Proposal server-state projections", () => {
  it("returns server-computed per-action gates verbatim", () => {
    const value = detail().gates;
    expect(approvalGate(value)).toBe(value.approvalAction);
    expect(releaseGate(value)).toBe(value.releaseAction);
    expect(cancelGate(value)).toBe(value.cancelAction);
  });
  it.each([
    { name: "no evidence", gates: {}, status: "DRAFT" as const, states: ["complete", "current", "pending", "pending"] },
    { name: "passing evidence", gates: { hasCurrentPassingCheck: true }, status: "REVIEWED" as const, states: ["complete", "complete", "current", "pending"] },
    { name: "current approval", gates: { hasCurrentPassingCheck: true, hasCurrentApproval: true }, status: "APPROVED" as const, states: ["complete", "complete", "complete", "current"] },
    { name: "released", gates: { hasCurrentPassingCheck: true, hasCurrentApproval: true, releasable: true }, status: "RELEASED" as const, states: ["complete", "complete", "complete", "complete"] },
  ])("renders $name from server data", ({ gates, status, states }) => {
    expect(progressSteps(detail(gates, status), gates.hasCurrentPassingCheck ? currentCheck : undefined).map((step) => step.state)).toEqual(states);
  });

  it("forces evidence and approval pending when the displayed server result is stale", () => {
    expect(progressSteps(detail({ hasCurrentPassingCheck: true, hasCurrentApproval: true }), { ...currentCheck, current: false }).map((step) => step.state)).toEqual(["complete", "current", "pending", "pending"]);
  });

  it("extracts persisted evidence without inventing missing values", () => {
    expect(evidenceView(currentCheck)).toEqual({ findings: [{ severity: "warning", unit: "point/1", message: "Review wording" }], model: "gemma", previewFidelity: "FAST" });
    expect(evidenceView({ ...currentCheck, kind: "deterministic", evidence: { preview: { fidelity: "FAST" } } })).toEqual({ findings: [], model: undefined, previewFidelity: "FAST" });
  });

  it("preserves explicit Proposal order", () => {
    expect(moveOrdered(["chg_a", "chg_b", "chg_c"], 2, -1)).toEqual(["chg_a", "chg_c", "chg_b"]);
    expect(moveOrdered(["chg_a", "chg_b"], 0, -1)).toEqual(["chg_a", "chg_b"]);
  });
});

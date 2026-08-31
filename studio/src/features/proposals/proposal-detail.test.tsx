import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { render, screen } from "@testing-library/react";
import type { Change, ProposalDetail as ProposalDetailData } from "../../api/types";
import { ProposalDetail } from "./proposal-detail";

const mocks = vi.hoisted(() => ({ query: {} as Record<string, unknown>, mutationError: undefined as Error | undefined }));

vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/reachability", () => ({ useSystemStatus: () => ({ state: "connected", inference: "disabled" }) }));
vi.mock("../../app/use-async", () => ({
  useQuery: () => mocks.query,
  useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: mocks.mutationError, result: undefined, reset: vi.fn() }),
}));

const change: Change = { id: "chg_one", ledgerId: "ldg_one", sequence: 7, unit: "point/one", action: "PUT", desired: { value: true }, baseFingerprint: "", desiredFingerprint: "sha256:desired", status: "READY", createdAt: "2026-08-31T00:00:00Z" };
const detail: ProposalDetailData = {
  proposal: { id: "pr_one", ledgerId: "ldg_one", title: "Safety review", hash: "sha256:current", status: "DRAFT", changeIds: [change.id], createdAt: "2026-08-31T00:00:00Z" },
  changes: [change],
  currentHeadReleaseId: "",
  gates: {
    hasCurrentPassingCheck: false,
    hasCurrentApproval: false,
    baseMatchesHead: true,
    releasable: false,
    reason: "A current passing evaluation is required.",
    approvalAction: { enabled: false, reason: "A current passing evaluation is required before approval." },
    releaseAction: { enabled: false, reason: "A current passing evaluation is required." },
  },
};

function queryState(overrides: Record<string, unknown> = {}) {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn(), ...overrides };
}

beforeEach(() => {
  mocks.mutationError = undefined;
  mocks.query = queryState({
    data: {
      detail,
      checks: [{ id: "chk_stale", proposalHash: "sha256:old", kind: "natural-language", passed: false, summary: "Review required", evidence: { findings: [{ severity: "error", unit: "point/one", message: "Contains unsupported claim" }], model: "gemma", previewFidelity: "FAST" }, createdAt: "2026-08-31T00:01:00Z", current: false }],
      approvals: [],
    },
  });
});

describe("ProposalDetail", () => {
  it("renders identity, ordered Changes, progress, stale evidence, findings, and server gate reasons", () => {
    const html = renderToStaticMarkup(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />);
    expect(html).toContain("Safety review");
    expect(html).toContain("sha256:cur…");
    expect(html).toContain("initial HEAD");
    for (const step of ["Changes", "Evidence", "Approval", "Release"]) expect(html).toContain(step);
    expect(html).toContain("Evidence was recorded for a different proposal hash and no longer applies.");
    expect(html).toContain("Contains unsupported claim");
    expect(html).toContain("Preview is an overlay summary, not a simulated query result.");
    expect(html).toContain("Natural-language evaluation is currently off. This stored check was recorded when inference was enabled.");
    expect(html).toContain("A current passing evaluation is required before approval.");
    expect(html).toContain("A current passing evaluation is required.");
    expect(html).toContain("Cancellation is available in GRF-212.");
  });

  it("renders loading, HTTP error, and stale populated detail states", () => {
    mocks.query = queryState({ loading: true });
    expect(renderToStaticMarkup(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />)).toContain("Loading Proposal detail");
    mocks.query = queryState({ error: new Error("Detail failed") });
    expect(renderToStaticMarkup(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />)).toContain("Detail failed");
    mocks.query = queryState({ data: { detail, checks: [], approvals: [] }, refetching: true });
    const stale = renderToStaticMarkup(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />);
    expect(stale).toContain("Safety review");
  });

  it("ignores stale approvals and retains the server release reason", () => {
    mocks.query = queryState({
      data: {
        detail: { ...detail, gates: { ...detail.gates, hasCurrentPassingCheck: true, approvalAction: { enabled: true, reason: "" }, reason: "A current approval is required.", releaseAction: { enabled: false, reason: "A current approval is required." } } },
        checks: [{ id: "chk_current", proposalHash: detail.proposal.hash, kind: "deterministic", passed: true, summary: "Passed", evidence: { preview: { fidelity: "FAST" } }, createdAt: "2026-08-31T00:01:00Z", current: true }],
        approvals: [{ id: "apr_stale", proposalHash: "sha256:old", actor: "reviewer", createdAt: "2026-08-31T00:02:00Z", current: false }],
      },
    });
    const html = renderToStaticMarkup(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />);
    expect(html).toContain("No current approval has been recorded.");
    expect(html).toContain("A current approval is required.");
  });

  it("renders server-authored disabled reasons beside disabled actions", () => {
    render(<ProposalDetail ledgerId="ldg_one" proposalId="pr_one" onUpdated={() => undefined} />);
    expect(screen.getByRole("button", { name: "Approve" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Approve" })).toHaveAttribute("title", detail.gates.approvalAction.reason);
    expect(screen.getByText(detail.gates.approvalAction.reason)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Release to Qdrant" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Release to Qdrant" })).toHaveAttribute("title", detail.gates.releaseAction.reason);
    expect(screen.getByText(detail.gates.releaseAction.reason)).toBeInTheDocument();
  });
});

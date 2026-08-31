import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { ApiError } from "../../api/client";
import type { ProposalDetail } from "../../api/types";
import { ReleasePanel } from "./release-panel";

const mocks = vi.hoisted(() => ({ error: undefined as Error | undefined }));
vi.mock("../../app/use-async", () => ({ useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: mocks.error, result: undefined, reset: vi.fn() }) }));

const detail: ProposalDetail = {
  proposal: { id: "pr_one", ledgerId: "ldg_one", title: "Release", hash: "sha256:one", status: "APPROVED", changeIds: ["chg_one", "chg_two"], createdAt: "2026-08-31T00:00:00Z" },
  changes: [
    { id: "chg_one", ledgerId: "ldg_one", sequence: 1, unit: "point/one", action: "PUT", desired: {}, baseFingerprint: "", desiredFingerprint: "sha256:one", status: "READY", createdAt: "2026-08-31T00:00:00Z" },
    { id: "chg_two", ledgerId: "ldg_one", sequence: 2, unit: "point/two", action: "DELETE", desired: null, baseFingerprint: "", desiredFingerprint: "sha256:two", status: "READY", createdAt: "2026-08-31T00:00:00Z" },
  ],
  currentHeadReleaseId: "",
  gates: { hasCurrentPassingCheck: true, hasCurrentApproval: true, baseMatchesHead: true, releasable: true, reason: "", approvalAction: { enabled: true, reason: "" }, releaseAction: { enabled: true, reason: "" } },
};

beforeEach(() => { mocks.error = undefined; });

describe("ReleasePanel", () => {
  it("requires the destructive confirmation with concrete consequences", () => {
    const html = renderToStaticMarkup(<ReleasePanel ledgerId="ldg_one" detail={detail} onRefresh={() => undefined} />);
    expect(html).toContain("Release to Qdrant?");
    expect(html).toContain("target collection will be mutated");
    expect(html).toContain("before-images will be retained");
    expect(html).toContain("2 affected units");
  });

  it("renders an HTTP 503 as a persistent recovery banner", () => {
    mocks.error = new ApiError("UNAVAILABLE", "Target apply failed; recovery is required.", 503, "http");
    const html = renderToStaticMarkup(<ReleasePanel ledgerId="ldg_one" detail={detail} onRefresh={() => undefined} />);
    expect(html).toContain("Target apply failed; recovery is required.");
    expect(html).toContain("Open Releases recovery");
    expect(html).toContain('href="#releases"');
  });

  it("does not misclassify transport failure as durable recovery", () => {
    mocks.error = new ApiError("UNAVAILABLE", "Cannot reach the Gyrifi runtime.", 0, "transport");
    const html = renderToStaticMarkup(<ReleasePanel ledgerId="ldg_one" detail={detail} onRefresh={() => undefined} />);
    expect(html).not.toContain("Open Releases recovery");
    expect(html).toContain("Cannot reach the Gyrifi runtime.");
  });
});

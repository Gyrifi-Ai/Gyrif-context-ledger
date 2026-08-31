import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { ApiError } from "../../api/client";
import type { Change } from "../../api/types";
import { CreateProposalDrawer, ProposalOrder } from "./create-proposal-drawer";

const mocks = vi.hoisted(() => ({ error: undefined as Error | undefined }));
vi.mock("../../app/use-async", () => ({ useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: mocks.error, result: undefined, reset: vi.fn() }) }));

const changes: Change[] = [
  { id: "chg_one", ledgerId: "ldg_one", sequence: 1, unit: "point/one", action: "PUT", desired: {}, baseFingerprint: "", desiredFingerprint: "sha256:one", status: "READY", createdAt: "2026-08-31T00:00:00Z" },
  { id: "chg_two", ledgerId: "ldg_one", sequence: 2, unit: "point/two", action: "DELETE", desired: null, baseFingerprint: "", desiredFingerprint: "sha256:two", status: "READY", createdAt: "2026-08-31T00:00:00Z" },
];

beforeEach(() => { mocks.error = undefined; });

describe("CreateProposalDrawer", () => {
  it("renders an ordered selectable READY-Change table", () => {
    const html = renderToStaticMarkup(<CreateProposalDrawer open ledgerId="ldg_one" changes={changes} onClose={() => undefined} onCreated={() => undefined} onConflict={() => undefined} />);
    expect(html).toContain("Selection order affects the Proposal hash.");
    expect(html).toContain("point/one");
    expect(html).toContain("point/two");
    expect(html).toContain('aria-label="Select chg_one"');
    const order = renderToStaticMarkup(<ProposalOrder changes={changes} orderedIds={changes.map((change) => change.id)} onChange={() => undefined} />);
    expect(order).toContain("1. point/one");
    expect(order).toContain("2. point/two");
    expect(order).toContain('aria-label="Move point/two up"');
  });

  it("surfaces creation errors verbatim", () => {
    mocks.error = new ApiError("CONFLICT", "One or more Changes are already in another active Proposal.", 409, "http");
    const html = renderToStaticMarkup(<CreateProposalDrawer open ledgerId="ldg_one" changes={changes} onClose={() => undefined} onCreated={() => undefined} onConflict={() => undefined} />);
    expect(html).toContain("One or more Changes are already in another active Proposal.");
  });
});

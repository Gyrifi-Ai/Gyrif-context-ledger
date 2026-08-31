import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { ApiError } from "../../api/client";
import { rollbackAffectedUnitCount } from "./release-timeline";
import { RollbackDialog } from "./rollback-dialog";
import type { Release, ReleaseIntent } from "../../api/types";

const mocks = vi.hoisted(() => ({ error: undefined as Error | undefined }));
vi.mock("../../app/use-async", () => ({ useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: mocks.error, result: undefined, reset: vi.fn() }) }));

const releases: Release[] = [
  { id: "rel_three", ledgerId: "ldg_one", proposalId: "pr_three", hash: "sha256:three", createdAt: "2026-08-31T03:00:00Z" },
  { id: "rel_two", ledgerId: "ldg_one", proposalId: "pr_two", hash: "sha256:two", createdAt: "2026-08-31T02:00:00Z" },
  { id: "rel_one", ledgerId: "ldg_one", proposalId: "pr_one", hash: "sha256:one", createdAt: "2026-08-31T01:00:00Z" },
];
const intents: ReleaseIntent[] = [
  { id: "intent_three", ledgerId: "ldg_one", proposalId: "pr_three", proposalHash: "sha256:three", status: "FINALIZED", plan: { operations: [{ unit: "shared", action: "PUT", expectedExists: true, desiredFingerprint: "sha256:a", beforeExists: true, hasBeforeImage: true }, { unit: "only-three", action: "DELETE", expectedExists: true, desiredFingerprint: "sha256:b", beforeExists: true, hasBeforeImage: true }] }, createdAt: releases[0].createdAt },
  { id: "intent_two", ledgerId: "ldg_one", proposalId: "pr_two", proposalHash: "sha256:two", status: "FINALIZED", plan: { operations: [{ unit: "shared", action: "PUT", expectedExists: true, desiredFingerprint: "sha256:c", beforeExists: true, hasBeforeImage: true }] }, createdAt: releases[1].createdAt },
];

beforeEach(() => { mocks.error = undefined; });

describe("RollbackDialog", () => {
  it("states all four forward-history consequences and the exact affected count", () => {
    const html = renderToStaticMarkup(<RollbackDialog open onClose={() => undefined} ledgerId="ldg_one" release={releases[2]} affectedUnitCount={2} onCreated={() => undefined} />);
    expect(html).toContain("new proposal");
    expect(html).toContain("does not rewind history");
    expect(html).toContain("2 units");
    expect(html).toContain("evaluated, approved, and released like any other");
    expect(html).toContain("HEAD will move forward");
    expect(html).toContain("2 affected units");
    expect(html).not.toContain("autofocus");
  });

  it("keeps server rollback failures verbatim inside the confirmation", () => {
    mocks.error = new ApiError("CONFLICT", "The selected Release is already HEAD.", 409, "http");
    let html = renderToStaticMarkup(<RollbackDialog open onClose={() => undefined} ledgerId="ldg_one" release={releases[2]} affectedUnitCount={2} onCreated={() => undefined} />);
    expect(html).toContain("The selected Release is already HEAD.");
    expect(html).toContain("Create rollback proposal");
    mocks.error = new ApiError("INTERNAL", "Retained rollback value is unavailable.", 500, "http");
    html = renderToStaticMarkup(<RollbackDialog open onClose={() => undefined} ledgerId="ldg_one" release={releases[2]} affectedUnitCount={2} onCreated={() => undefined} />);
    expect(html).toContain("Retained rollback value is unavailable.");
  });

  it("counts unique units from every release newer than the target and refuses missing plans", () => {
    expect(rollbackAffectedUnitCount(releases, "rel_one", intents)).toBe(2);
    expect(rollbackAffectedUnitCount(releases, "rel_two", intents)).toBe(2);
    expect(rollbackAffectedUnitCount(releases, "rel_three", intents)).toBe(0);
    expect(rollbackAffectedUnitCount(releases, "missing", intents)).toBeUndefined();
    expect(rollbackAffectedUnitCount(releases, "rel_one", intents.slice(1))).toBeUndefined();
  });
});
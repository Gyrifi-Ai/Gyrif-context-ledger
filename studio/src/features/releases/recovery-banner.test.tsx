import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { ReleaseIntent } from "../../api/types";
import { RecoveryBanner } from "./recovery-banner";

const mocks = vi.hoisted(() => ({ error: undefined as Error | undefined }));
vi.mock("../../app/use-async", () => ({ useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: mocks.error, result: undefined, reset: vi.fn() }) }));

const intent: ReleaseIntent = { id: "intent_recovery_123", ledgerId: "ldg_one", proposalId: "pr_one", proposalHash: "sha256:proposal", status: "RECOVERY_REQUIRED", plan: { operations: [{ unit: "point/one", action: "PUT", expectedExists: true, expectedFingerprint: "sha256:before", desiredFingerprint: "sha256:desired", beforeExists: true, hasBeforeImage: true }] }, createdAt: "2026-08-31T00:00:00Z" };

beforeEach(() => { mocks.error = undefined; });

describe("RecoveryBanner", () => {
  it("is absent without affected intents", () => {
    expect(renderToStaticMarkup(<RecoveryBanner ledgerId="ldg_one" intents={[]} onUpdated={() => undefined} />)).toBe("");
  });

  it("shows the count, inspect surface, intent plan, and recovery actions", () => {
    const html = renderToStaticMarkup(<RecoveryBanner ledgerId="ldg_one" intents={[intent, { ...intent, id: "intent_two" }]} onUpdated={() => undefined} />);
    expect(html).toContain("2 release intents require recovery.");
    expect(html).toContain("Inspect");
    expect(html).toContain("RECOVERY_REQUIRED");
    expect(html).toContain("point/one");
    expect(html).toContain("Retry verification");
    expect(html).toContain("Mark resolved");
  });
});
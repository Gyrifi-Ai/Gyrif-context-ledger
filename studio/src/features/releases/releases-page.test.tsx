import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Proposal, Release, ReleaseIntent } from "../../api/types";
import { ReleasesPage, RollbackSuccess } from "./releases-page";

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
  appState: { ledgerId: "ldg_one", openLedgerSwitcher: vi.fn() },
  mutation: { run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: undefined, result: undefined, reset: vi.fn() },
}));

vi.mock("../../app/providers", () => ({ useAppState: () => mocks.appState }));
vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/use-async", () => ({ useQuery: () => mocks.query, useMutation: () => mocks.mutation }));

const proposal: Proposal = { id: "pr_refund", ledgerId: "ldg_one", title: "August refund policy refresh", hash: "sha256:proposal", status: "RELEASED", changeIds: ["chg_one"], createdAt: "2026-08-31T00:00:00Z" };
const releases: Release[] = [
  { id: "rel_newest_123456", ledgerId: "ldg_one", proposalId: proposal.id, hash: "sha256:newest-release", createdAt: "2026-08-31T02:00:00Z" },
  { id: "rel_older_1234567", ledgerId: "ldg_one", proposalId: "pr_older", hash: "sha256:older-release", createdAt: "2026-08-31T01:00:00Z" },
];
const intents: ReleaseIntent[] = releases.map((release, index) => ({ id: `intent_${index}`, ledgerId: "ldg_one", proposalId: release.proposalId, proposalHash: "sha256:proposal", status: "FINALIZED", plan: { operations: [{ unit: `point/${index}`, action: "PUT", expectedExists: false, desiredFingerprint: `sha256:${index}`, beforeExists: false, hasBeforeImage: false }] }, createdAt: release.createdAt }));

function query(overrides: Record<string, unknown> = {}) {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn(), ...overrides };
}

beforeEach(() => {
  mocks.appState.ledgerId = "ldg_one";
  mocks.query = query({ data: { releases, proposals: [proposal], intents } });
});

describe("ReleasesPage", () => {
  it("renders the header and newest-first timeline with HEAD and rollback actions", () => {
    const html = renderToStaticMarkup(<ReleasesPage />);
    expect(html).toContain("IMMUTABLE HISTORY");
    expect(html).toContain("Every release was applied to the target and verified before it was recorded.");
    expect(html.indexOf("rel_newest")).toBeLessThan(html.indexOf("rel_older_"));
    expect(html).toContain("August refund policy refresh");
    expect(html).toContain("HEAD");
    expect(html.match(/View plan/g)).toHaveLength(2);
    expect(html.match(/Roll back to here/g)).toHaveLength(1);
  });

  it("renders loading, empty, error, and stale states", () => {
    mocks.query = query({ loading: true });
    expect(renderToStaticMarkup(<ReleasesPage />)).toContain("Loading Releases");
    mocks.query = query({ data: { releases: [], proposals: [], intents: [] } });
    const empty = renderToStaticMarkup(<ReleasesPage />);
    expect(empty).toContain("No Releases yet");
    expect(empty).toContain('href="#proposals"');
    mocks.query = query({ error: new Error("Release history failed") });
    expect(renderToStaticMarkup(<ReleasesPage />)).toContain("Release history failed");
    mocks.query = query({ data: { releases, proposals: [proposal], intents }, refetching: true });
    expect(renderToStaticMarkup(<ReleasesPage />)).toContain("rel_newest");
  });

  it("links a created rollback proposal directly into review", () => {
    const html = renderToStaticMarkup(<RollbackSuccess proposal={{ ...proposal, id: "pr_rollback", title: "Restore state from rel_older" }} />);
    expect(html).toContain("Rollback proposal created: Restore state from rel_older");
    expect(html).toContain('href="#proposals/pr_rollback"');
    expect(html).toContain("Review proposal");
  });

  it("opens rollback confirmation with all forward-history consequences", async () => {
    const user = userEvent.setup();
    render(<ReleasesPage />);
    await user.click(screen.getByRole("button", { name: "Roll back to here" }));
    const dialog = screen.getByRole("dialog", { name: "Create rollback proposal?" });
    expect(dialog).toHaveTextContent("new proposal");
    expect(dialog).toHaveTextContent("does not rewind history");
    expect(dialog).toHaveTextContent("1 units will be restored");
    expect(dialog).toHaveTextContent("evaluated, approved, and released like any other");
    expect(dialog).toHaveTextContent("HEAD will move forward");
  });

  it("shows recovery only for recovery-required intents and opens inspection", async () => {
    const user = userEvent.setup();
    const recovery = { ...intents[0], id: "intent_recovery", status: "RECOVERY_REQUIRED" as const };
    mocks.query = query({ data: { releases, proposals: [proposal], intents: [...intents, recovery] } });
    const { rerender } = render(<ReleasesPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("1 release intent requires recovery.");
    await user.click(screen.getByRole("button", { name: "Inspect" }));
    expect(screen.getByRole("dialog", { name: "Release recovery" })).toHaveTextContent("Retry verification");
    mocks.query = query({ data: { releases, proposals: [proposal], intents } });
    rerender(<ReleasesPage />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("disables rollback when a newer Release plan is unavailable", () => {
    mocks.query = query({ data: { releases, proposals: [proposal], intents: intents.slice(1) } });
    render(<ReleasesPage />);
    const rollback = screen.getByRole("button", { name: "Roll back to here" });
    expect(rollback).toBeDisabled();
    expect(rollback).toHaveAttribute("title", "Rollback plan is unavailable.");
  });
});
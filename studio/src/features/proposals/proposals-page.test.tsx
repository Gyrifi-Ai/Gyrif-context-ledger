import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { Change, Proposal } from "../../api/types";
import { ProposalsPage } from "./proposals-page";

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
  appState: { ledgerId: "ldg_one", openLedgerSwitcher: vi.fn() },
}));
vi.mock("../../app/providers", () => ({ useAppState: () => mocks.appState }));
vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/use-async", () => ({
  useQuery: () => mocks.query,
  useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: undefined, result: undefined, reset: vi.fn() }),
}));

const proposal: Proposal = { id: "pr_one", ledgerId: "ldg_one", title: "Safety review", hash: "sha256:one", status: "DRAFT", changeIds: ["chg_one"], createdAt: "2026-08-31T00:00:00Z" };
const change: Change = { id: "chg_one", ledgerId: "ldg_one", sequence: 1, unit: "point/one", action: "PUT", desired: {}, baseFingerprint: "", desiredFingerprint: "sha256:desired", status: "READY", createdAt: "2026-08-31T00:00:00Z" };

function queryState(overrides: Record<string, unknown> = {}) {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn(), ...overrides };
}

beforeEach(() => {
  mocks.appState.ledgerId = "ldg_one";
  mocks.appState.openLedgerSwitcher.mockReset();
  mocks.query = queryState({ data: { proposals: [proposal], readyChanges: [change] } });
});

describe("ProposalsPage", () => {
  it("renders the linkable review queue and page-owned action", () => {
    const html = renderToStaticMarkup(<ProposalsPage />);
    expect(html).toContain("CONTEXT PRs");
    expect(html).toContain("New proposal");
    expect(html).toContain("Safety review");
    expect(html).toContain("1 changes");
    expect(html).toContain("Select a Proposal");
  });

  it("renders list loading, empty, error, and stale populated states", () => {
    mocks.query = queryState({ loading: true });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("Loading Proposals");
    mocks.query = queryState({ data: { proposals: [], readyChanges: [] } });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("No Proposals");
    mocks.query = queryState({ error: new Error("Queue failed") });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("Queue failed");
    mocks.query = queryState({ data: { proposals: [proposal], readyChanges: [change] }, refetching: true });
    const stale = renderToStaticMarkup(<ProposalsPage />);
    expect(stale).toContain("gy-is-refetching");
    expect(stale).toContain("Safety review");
  });

  it("renders the shared ledger-switcher action when unscoped", () => {
    mocks.appState.ledgerId = "";
    const html = renderToStaticMarkup(<ProposalsPage />);
    expect(html).toContain("Select a ledger");
    expect(html).toContain("Select ledger");
  });
});

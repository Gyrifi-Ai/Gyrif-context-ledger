import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { render, screen } from "@testing-library/react";
import type { Change, Proposal } from "../../api/types";
import { ProposalsPage } from "./proposals-page";

const mocks = vi.hoisted(() => ({
  proposalsQuery: {} as Record<string, unknown>,
  changesQuery: {} as Record<string, unknown>,
  appState: { ledgerId: "ldg_one", openLedgerSwitcher: vi.fn() },
}));
vi.mock("../../app/providers", () => ({ useAppState: () => mocks.appState }));
vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/use-async", () => ({
  useMutation: () => ({ run: vi.fn(), pending: false, blocked: false, disabledReason: undefined, error: undefined, result: undefined, reset: vi.fn() }),
}));
vi.mock("../../app/use-paginated-query", () => ({ usePaginatedQuery: (key: string) => key === "proposals" ? mocks.proposalsQuery : mocks.changesQuery }));

const proposal: Proposal = { id: "pr_one", ledgerId: "ldg_one", title: "Safety review", hash: "sha256:one", status: "DRAFT", changeIds: ["chg_one"], createdAt: "2026-08-31T00:00:00Z" };
const change: Change = { id: "chg_one", ledgerId: "ldg_one", sequence: 1, unit: "point/one", action: "PUT", desired: {}, baseFingerprint: "", desiredFingerprint: "sha256:desired", status: "READY", createdAt: "2026-08-31T00:00:00Z" };

function queryState(overrides: Record<string, unknown> = {}) {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn(), ...overrides };
}

beforeEach(() => {
  mocks.appState.ledgerId = "ldg_one";
  mocks.appState.openLedgerSwitcher.mockReset();
  mocks.proposalsQuery = queryState({ data: [proposal], nextCursor: undefined, loadingMore: false, loadMoreError: undefined, loadMore: vi.fn() });
  mocks.changesQuery = queryState({ data: [change], nextCursor: undefined, loadingMore: false, loadMoreError: undefined, loadMore: vi.fn() });
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
    mocks.proposalsQuery = queryState({ loading: true });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("Loading Proposals");
    mocks.proposalsQuery = queryState({ data: [] });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("No Proposals");
    mocks.proposalsQuery = queryState({ error: new Error("Queue failed") });
    expect(renderToStaticMarkup(<ProposalsPage />)).toContain("Queue failed");
    mocks.proposalsQuery = queryState({ data: [proposal], refetching: true });
    const stale = renderToStaticMarkup(<ProposalsPage />);
    expect(stale).toContain("Safety review");
  });

  it("renders the shared ledger-switcher action when unscoped", () => {
    mocks.appState.ledgerId = "";
    const html = renderToStaticMarkup(<ProposalsPage />);
    expect(html).toContain("Select a ledger");
    expect(html).toContain("Select ledger");
  });

  it("disables Load more while the next Proposal page is loading", () => {
    mocks.proposalsQuery = queryState({ data: [proposal], nextCursor: "next", loadingMore: true, loadMore: vi.fn() });
    render(<ProposalsPage />);
    expect(screen.getByRole("button", { name: "Load more" })).toBeDisabled();
  });
});

import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { ApiError } from "../../api/client";
import type { Change } from "../../api/types";
import { ChangesPage } from "./changes-page";
import { SelectionActionBar } from "./selection-action-bar";

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
  mutationIndex: 0,
  mutations: [
    { run: vi.fn(), pending: false, blocked: false, disabledReason: undefined as string | undefined, error: undefined as Error | undefined, result: undefined, reset: vi.fn() },
    { run: vi.fn(), pending: false, blocked: false, disabledReason: undefined as string | undefined, error: undefined as Error | undefined, result: undefined, reset: vi.fn() },
  ],
  appState: { ledgerId: "ldg_one", openLedgerSwitcher: vi.fn() },
}));

vi.mock("../../app/providers", () => ({ useAppState: () => mocks.appState }));
vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/use-async", () => ({
  useQuery: () => mocks.query,
  useMutation: () => mocks.mutations[mocks.mutationIndex++],
}));

function queryState(overrides: Record<string, unknown> = {}) {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn(), ...overrides };
}

const ready: Change = {
  id: "chg_ready",
  ledgerId: "ldg_one",
  sequence: 42,
  unit: "point/ready",
  action: "PUT",
  desired: { value: true },
  baseFingerprint: "",
  desiredFingerprint: "sha256:123456789abcdef",
  status: "READY",
  createdAt: "2026-08-31T11:58:00Z",
};
const released: Change = { ...ready, id: "chg_released", sequence: 41, unit: "point/released", action: "DELETE", desired: null, status: "RELEASED" };

beforeEach(() => {
  mocks.mutationIndex = 0;
  mocks.appState.ledgerId = "ldg_one";
  mocks.appState.openLedgerSwitcher.mockReset();
  mocks.query = queryState({ data: [ready, released] });
  for (const mutation of mocks.mutations) {
    mutation.run.mockReset();
    mutation.reset.mockReset();
    mutation.pending = false;
    mutation.blocked = false;
    mutation.disabledReason = undefined;
    mutation.error = undefined;
    mutation.result = undefined;
  }
});

describe("ChangesPage", () => {
  it("renders the stats, seven-column selectable table, and absolute timestamps", () => {
    const html = renderToStaticMarkup(<ChangesPage />);
    expect(html).toContain("DURABLE INBOX");
    expect(html).toContain("Submit change");
    for (const header of ["SEQ", "UNIT", "ACTION", "DESIRED FINGERPRINT", "STATUS", "AGE"]) expect(html).toContain(header);
    expect(html).toContain("point/ready");
    expect(html).toContain("point/released");
    expect(html).toContain('title="2026-08-31T11:58:00Z"');
    expect(html).toContain("Server-side bounds and filtering are tracked by GRF-214.");
    expect(html.match(/>1<\/p>/g)?.length).toBeGreaterThanOrEqual(2);
  });

  it("disables selection for non-READY Changes with an explanation", () => {
    const html = renderToStaticMarkup(<ChangesPage />);
    expect(html).toContain('aria-label="Select chg_ready"');
    expect(html).toContain('aria-label="Select chg_released"');
    expect(html).toContain('title="RELEASED Changes cannot be added to a Proposal."');
    expect(html).toMatch(/aria-label="Select chg_released"[^>]*disabled/);
  });

  it("renders loading, empty, and error states", () => {
    mocks.query = queryState({ loading: true });
    const loading = renderToStaticMarkup(<ChangesPage />);
    expect(loading.match(/aria-busy="true"/g)?.length).toBeGreaterThanOrEqual(3);

    mocks.mutationIndex = 0;
    mocks.query = queryState({ data: [] });
    const empty = renderToStaticMarkup(<ChangesPage />);
    expect(empty).toContain("No Changes yet");
    expect(empty).toContain("/api/v1/ledgers/{id}/changes");

    mocks.mutationIndex = 0;
    mocks.query = queryState({ error: new Error("Inbox failed") });
    const error = renderToStaticMarkup(<ChangesPage />);
    expect(error).toContain("Inbox failed");
    expect(error).toContain("Retry");
  });

  it("keeps populated rows visible and dimmed while refetching", () => {
    mocks.query = queryState({ data: [ready], refetching: true });
    const html = renderToStaticMarkup(<ChangesPage />);
    expect(html).toContain("gy-is-refetching");
    expect(html).toContain("point/ready");
  });

  it("renders the selection action bar only with a selected count", () => {
    expect(renderToStaticMarkup(<SelectionActionBar count={0} onClear={() => undefined} onCreateProposal={() => undefined} />)).toBe("");
    const html = renderToStaticMarkup(<SelectionActionBar count={2} onClear={() => undefined} onCreateProposal={() => undefined} />);
    expect(html).toContain("2 selected");
    expect(html).toContain("Clear");
    expect(html).toContain("Create proposal");
  });

  it("renders the select-ledger action when no ledger is selected", () => {
    mocks.appState.ledgerId = "";
    const html = renderToStaticMarkup(<ChangesPage />);
    expect(html).toContain("Select a ledger");
    expect(html).toContain("Select ledger");
  });

  it("renders proposal conflicts verbatim", () => {
    mocks.mutations[1].error = new ApiError("CONFLICT", "One or more Changes are already in another active Proposal.", 409, "http");
    const html = renderToStaticMarkup(<ChangesPage />);
    expect(html).toContain("One or more Changes are already in another active Proposal.");
  });
});
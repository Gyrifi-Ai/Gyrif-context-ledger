import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiError } from "../../api/client";
import type { Change, Ledger } from "../../api/types";
import { LedgersPage } from "./ledgers-page";
import { countReadyChanges, validateLedgerForm } from "./ledgers-page.logic";

const mocks = vi.hoisted(() => ({
  queryStates: new Map<string, Record<string, unknown>>(),
  mutation: {
    run: vi.fn(), pending: false, blocked: false, disabledReason: undefined as string | undefined,
    error: undefined as Error | undefined, result: undefined, reset: vi.fn(),
  },
  appState: {
    ledgerId: "",
    refreshLedgers: vi.fn(),
    setLedgerId: vi.fn(),
  },
}));

vi.mock("../../app/providers", () => ({ useAppState: () => mocks.appState }));
vi.mock("../../app/use-ledger-events", () => ({ useLedgerEvents: vi.fn() }));
vi.mock("../../app/use-async", () => ({
  useQuery: (key: string) => mocks.queryStates.get(key) ?? queryState({ data: 0 }),
  useMutation: () => mocks.mutation,
}));

function queryState(overrides: Record<string, unknown> = {}) {
  return {
    data: undefined,
    error: undefined,
    loading: false,
    refetching: false,
    unavailable: false,
    refetch: vi.fn(),
    ...overrides,
  };
}

const ledger: Ledger = {
  id: "ldg_123456789abcdef",
  name: "product-docs",
  description: "Policies and product documentation used by support.",
  createdAt: "2026-08-31T00:00:00Z",
};

beforeEach(() => {
  mocks.queryStates.clear();
  mocks.appState.ledgerId = "";
  mocks.appState.refreshLedgers.mockReset();
  mocks.appState.setLedgerId.mockReset();
  mocks.mutation.run.mockReset();
  mocks.mutation.reset.mockReset();
  mocks.mutation.pending = false;
  mocks.mutation.blocked = false;
  mocks.mutation.disabledReason = undefined;
  mocks.mutation.error = undefined;
});

describe("LedgersPage", () => {
  it("renders three matching skeleton cards before rendering populated cards", () => {
    mocks.queryStates.set("ledgers", queryState({ loading: true }));
    const loading = renderToStaticMarkup(<LedgersPage />);
    expect(loading).toContain('aria-label="Loading ledgers"');
    expect(loading).toContain('aria-busy="true"');

    mocks.appState.ledgerId = ledger.id;
    mocks.queryStates.set("ledgers", queryState({ data: [ledger] }));
    mocks.queryStates.set(`ledger-ready-count-${ledger.id}`, queryState({ data: 2 }));
    mocks.queryStates.set(`ledger-release-count-${ledger.id}`, queryState({ data: 4 }));
    const populated = renderToStaticMarkup(<LedgersPage />);

    expect(populated).toContain("product-docs");
    expect(populated).toContain("2 ready");
    expect(populated).toContain("4 releases");
    expect(populated).toContain("ACTIVE");
    expect(populated).toContain('type="button"');
    expect(populated).toContain("ldg_123456…");
  });

  it("renders the empty state with its creation action", () => {
    mocks.queryStates.set("ledgers", queryState({ data: [] }));
    const html = renderToStaticMarkup(<LedgersPage />);
    expect(html).toContain("No ledgers yet");
    expect(html).toContain("Create your first ledger");
    expect(html).toContain("governed namespace");
  });

  it("renders a list error with Retry", () => {
    mocks.queryStates.set("ledgers", queryState({ error: new Error("Ledger list failed") }));
    const html = renderToStaticMarkup(<LedgersPage />);
    expect(html).toContain("Ledger list failed");
    expect(html).toContain("Retry");
  });

  it("surfaces duplicate-name conflicts only on the name field", () => {
    mocks.queryStates.set("ledgers", queryState({ data: [] }));
    mocks.mutation.error = new ApiError("CONFLICT", "A ledger with that name already exists.", 409, "http");
    const html = renderToStaticMarkup(<LedgersPage />);
    expect(html.match(/A ledger with that name already exists\./g)).toHaveLength(1);
    expect(html).not.toContain("Unable to create ledger");
  });

  it("keeps a card usable when either count is unavailable", () => {
    mocks.queryStates.set("ledgers", queryState({ data: [ledger] }));
    mocks.queryStates.set(`ledger-ready-count-${ledger.id}`, queryState({ error: new Error("failed") }));
    mocks.queryStates.set(`ledger-release-count-${ledger.id}`, queryState({ data: 1 }));
    const html = renderToStaticMarkup(<LedgersPage />);
    expect(html).toContain("product-docs");
    expect(html).toContain("— ready");
    expect(html).toContain("1 releases");
  });

  it("validates creation in the drawer and submits trimmed input", async () => {
    const user = userEvent.setup();
    mocks.queryStates.set("ledgers", queryState({ data: [] }));
    render(<LedgersPage />);
    await user.click(screen.getByRole("button", { name: "Create your first ledger" }));
    expect(screen.getByRole("dialog", { name: "Create ledger" })).toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Name" }), "   ");
    await user.click(screen.getByRole("button", { name: "Create ledger" }));
    expect(screen.getByRole("alert")).toHaveTextContent("A ledger name is required.");
    await user.clear(screen.getByRole("textbox", { name: "Name" }));
    await user.type(screen.getByRole("textbox", { name: "Name" }), "  support-kb  ");
    await user.type(screen.getByRole("textbox", { name: "Description" }), "Support answers");
    await user.click(screen.getByRole("button", { name: "Create ledger" }));
    expect(mocks.mutation.run).toHaveBeenCalledWith({ ledgerName: "support-kb", ledgerDescription: "Support answers" });
  });

  it("switches the active ledger from a rendered card", async () => {
    const user = userEvent.setup();
    mocks.queryStates.set("ledgers", queryState({ data: [ledger] }));
    render(<LedgersPage />);
    await user.click(screen.getByRole("button", { name: /product-docs/ }));
    expect(mocks.appState.setLedgerId).toHaveBeenCalledWith(ledger.id);
    expect(screen.getByRole("status")).toHaveTextContent("Now governing product-docs");
  });

  it("disables creation while the Runtime is unreachable", async () => {
    const user = userEvent.setup();
    mocks.queryStates.set("ledgers", queryState({ data: [] }));
    mocks.mutation.blocked = true;
    mocks.mutation.disabledReason = "Cannot reach the Gyrifi runtime.";
    render(<LedgersPage />);
    await user.click(screen.getByRole("button", { name: "Create your first ledger" }));
    const create = screen.getByRole("button", { name: "Create ledger" });
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "Cannot reach the Gyrifi runtime.");
  });
});

describe("ledger helpers", () => {
  it("validates trimmed names and description limits", () => {
    expect(validateLedgerForm("   ", "").name).toBe("A ledger name is required.");
    expect(validateLedgerForm("a".repeat(121), "").name).toBe("Name must be 120 characters or fewer.");
    expect(validateLedgerForm("valid", "a".repeat(501)).description).toBe("Description must be 500 characters or fewer.");
    expect(validateLedgerForm("  valid  ", "optional")).toEqual({ name: undefined, description: undefined });
  });

  it("counts only READY changes", () => {
    const changes = ["READY", "ACCEPTED", "READY", "RELEASED"].map((status, index) => ({ id: `${index}`, status })) as Change[];
    expect(countReadyChanges(changes)).toBe(2);
  });
});

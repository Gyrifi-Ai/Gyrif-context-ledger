import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Ledger } from "../../api/types";
import type { RuntimeHealth } from "../../app/reachability";
import { HeadChip } from "./head-chip";
import { LedgerSwitcher } from "./ledger-switcher";
import { Nav } from "./nav";
import { RuntimeStatus } from "./runtime-status";

const mocks = vi.hoisted(() => ({
  app: { ledger: null as Ledger | null, ledgers: [] as Ledger[], ledgerSwitcherRequest: 0, setLedgerId: vi.fn() },
  query: { data: undefined as { id: string }[] | undefined, error: undefined as Error | undefined, loading: false, refetching: false, unavailable: false, refetch: vi.fn() },
  reachability: { health: { state: "connected", version: "1.0", commit: "abc123", buildDate: "2026-09-01T00:00:00Z", inference: "disabled" } as RuntimeHealth, streamState: "open", streamExhausted: false, reconnectStream: vi.fn() },
}));

vi.mock("../../app/providers", () => ({ useAppState: () => mocks.app }));
vi.mock("../../app/use-async", () => ({ useQuery: () => mocks.query }));
vi.mock("../../app/reachability", () => ({ useReachability: () => mocks.reachability }));

const ledgers: Ledger[] = [
  { id: "ldg_one", name: "Product docs", description: "", createdAt: "2026-08-31T00:00:00Z" },
  { id: "ldg_two", name: "Support KB", description: "", createdAt: "2026-08-31T00:00:00Z" },
];

beforeEach(() => {
  mocks.app.ledger = ledgers[0];
  mocks.app.ledgers = ledgers;
  mocks.app.ledgerSwitcherRequest = 0;
  mocks.app.setLedgerId.mockReset();
  mocks.query.data = [];
  mocks.query.error = undefined;
  mocks.reachability.health = { state: "connected", version: "1.0", commit: "abc123", buildDate: "2026-09-01T00:00:00Z", inference: "disabled" };
  mocks.reachability.streamState = "open";
  mocks.reachability.streamExhausted = false;
  mocks.reachability.reconnectStream.mockReset();
});

describe("shell controls", () => {
  it("disables scoped destinations until a ledger is selected", () => {
    const { rerender } = render(<Nav route="ledgers" ledgerId="" />);
    expect(screen.getByRole("link", { name: "Ledgers" })).toHaveAttribute("href", "#ledgers");
    expect(screen.getByText("Changes").closest("a")).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByText("Changes").closest("a")).not.toHaveAttribute("href");
    rerender(<Nav route="changes" ledgerId="ldg_one" />);
    expect(screen.getByRole("link", { name: "Changes" })).toHaveAttribute("href", "#changes");
  });

  it("filters and chooses ledgers by keyboard", async () => {
    const user = userEvent.setup();
    render(<LedgerSwitcher />);
    await user.click(screen.getByRole("button", { name: /Product docs/ }));
    const filter = screen.getByPlaceholderText("Filter ledgers");
    await user.type(filter, "Support");
    expect(screen.getByRole("option", { name: "Support KB" })).toBeInTheDocument();
    await user.keyboard("{Enter}");
    expect(mocks.app.setLedgerId).toHaveBeenCalledWith("ldg_two");
  });

  it("reports an empty switcher result and closes on Escape", async () => {
    const user = userEvent.setup();
    render(<LedgerSwitcher />);
    await user.click(screen.getByRole("button", { name: /Product docs/ }));
    await user.type(screen.getByPlaceholderText("Filter ledgers"), "missing");
    expect(screen.getByText("No ledgers match.")).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("renders HEAD, initial history, and errors", () => {
    mocks.query.data = [{ id: "rel_123456789" }];
    const { rerender } = render(<HeadChip ledgerId="ldg_one" />);
    expect(screen.getByRole("button", { name: /HEAD · rel_123456/ })).toBeInTheDocument();
    mocks.query.data = [];
    rerender(<HeadChip ledgerId="ldg_one" />);
    expect(screen.getByText("No releases yet")).toBeInTheDocument();
    mocks.query.error = new Error("HEAD failed");
    rerender(<HeadChip ledgerId="ldg_one" />);
    expect(screen.getByRole("alert")).toHaveTextContent("HEAD failed");
    mocks.query.error = undefined;
    rerender(<HeadChip ledgerId="" />);
    expect(screen.getByText("No releases yet")).toBeInTheDocument();
  });

  it("renders runtime and event-stream states and reconnects", async () => {
    const user = userEvent.setup();
    const { rerender } = render(<RuntimeStatus />);
    expect(screen.getByText("Connected")).toHaveAttribute("title", "Runtime 1.0 (abc123, 2026-09-01T00:00:00Z) · inference disabled");
    expect(screen.getByLabelText("Runtime version")).toHaveTextContent("1.0");
    mocks.reachability.health = { state: "degraded" };
    rerender(<RuntimeStatus />);
    expect(screen.getByText("Degraded")).toBeInTheDocument();
    mocks.reachability.health = { state: "offline" };
    rerender(<RuntimeStatus />);
    expect(screen.getByText("Offline")).toHaveAttribute("title", "Runtime unreachable");
    mocks.reachability.health = { state: "connected" };
    mocks.reachability.streamExhausted = true;
    rerender(<RuntimeStatus />);
    await user.click(screen.getByRole("button", { name: "Reconnect" }));
    expect(mocks.reachability.reconnectStream).toHaveBeenCalledTimes(1);
  });
});

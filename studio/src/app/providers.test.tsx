import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { mockApi } from "../test/api-mock";
import { renderWithProviders } from "../test/render";
import { useAppState } from "./providers";

function Consumer() {
  const state = useAppState();
  return <><p>{state.ledger?.name ?? "No active ledger"}</p><p>{state.ledgers.length} ledgers</p><button onClick={() => state.setLedgerId("ldg_two")}>Select support</button><button onClick={() => void state.refreshLedgers()}>Refresh ledgers</button><button onClick={state.openLedgerSwitcher}>Open switcher {state.ledgerSwitcherRequest}</button></>;
}

beforeEach(() => localStorage.clear());

describe("Providers", () => {
  it("loads ledgers, persists selection, refreshes, and requests the switcher", async () => {
    mockApi.ledgers.mockResolvedValue({ items: [
      { id: "ldg_one", name: "Product docs", description: "", createdAt: "2026-08-31T00:00:00Z" },
      { id: "ldg_two", name: "Support KB", description: "", createdAt: "2026-08-31T00:00:00Z" },
    ] });
    const { user } = renderWithProviders(<Consumer />);
    expect(await screen.findByText("2 ledgers")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Select support" }));
    expect(screen.getByText("Support KB")).toBeInTheDocument();
    expect(localStorage.getItem("gyrifi.ledger")).toBe("ldg_two");
    await user.click(screen.getByRole("button", { name: "Refresh ledgers" }));
    await waitFor(() => expect(mockApi.ledgers).toHaveBeenCalledTimes(2));
    await user.click(screen.getByRole("button", { name: /Open switcher/ }));
    expect(screen.getByRole("button", { name: "Open switcher 1" })).toBeInTheDocument();
  });
});

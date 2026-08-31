import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useLedgerEvents } from "./use-ledger-events";

const mocks = vi.hoisted(() => ({ register: vi.fn(), unregister: vi.fn() }));
vi.mock("./reachability", () => ({ useReachability: () => ({ registerInvalidation: mocks.register }) }));

describe("useLedgerEvents", () => {
  it("registers the current ledger callback and unregisters on change", () => {
    mocks.register.mockImplementation(() => mocks.unregister);
    const first = vi.fn();
    const second = vi.fn();
    const { rerender, unmount } = renderHook(({ ledgerId, callback }) => useLedgerEvents(ledgerId, callback), { initialProps: { ledgerId: "ldg_one" as string | undefined, callback: first } });
    expect(mocks.register).toHaveBeenCalledWith("ldg_one", first);
    rerender({ ledgerId: "ldg_two", callback: second });
    expect(mocks.unregister).toHaveBeenCalledTimes(1);
    expect(mocks.register).toHaveBeenLastCalledWith("ldg_two", second);
    unmount();
    expect(mocks.unregister).toHaveBeenCalledTimes(2);
  });
});

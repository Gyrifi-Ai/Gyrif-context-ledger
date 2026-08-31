import { act, render, renderHook, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../api/client";
import { ErrorState } from "../ui/feedback/error-state";
import { useMutation, useQuery } from "./use-async";

const reachability = vi.hoisted(() => ({ unreachable: false }));
vi.mock("./reachability", () => ({
  runtimeUnavailableMessage: "Cannot reach the Gyrifi runtime. Displayed data may be out of date.",
  useReachability: () => reachability,
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((accept, fail) => { resolve = accept; reject = fail; });
  return { promise, resolve, reject };
}

beforeEach(() => { reachability.unreachable = false; });

describe("useQuery", () => {
  it("transitions from loading to success", async () => {
    const request = deferred<string[]>();
    const load = vi.fn(() => request.promise);
    const { result } = renderHook(() => useQuery("items", load, []));
    expect(result.current.loading).toBe(true);
    expect(load).toHaveBeenCalledTimes(1);
    await act(async () => request.resolve(["ready"]));
    expect(result.current.data).toEqual(["ready"]);
    expect(result.current.loading).toBe(false);
  });

  it("retains data during refetch and propagates HTTP errors", async () => {
    const second = deferred<string[]>();
    const load = vi.fn()
      .mockResolvedValueOnce(["current"])
      .mockImplementationOnce(() => second.promise);
    const { result } = renderHook(() => useQuery("items", load, []));
    await waitFor(() => expect(result.current.data).toEqual(["current"]));
    act(() => result.current.refetch());
    expect(result.current.data).toEqual(["current"]);
    expect(result.current.refetching).toBe(true);
    await act(async () => second.reject(new ApiError("CONFLICT", "Server rejected refresh", 409, "http")));
    expect(result.current.data).toEqual(["current"]);
    expect(result.current.error).toMatchObject({ code: "CONFLICT", message: "Server rejected refresh" });
    render(<ErrorState message={result.current.error!.message} onRetry={result.current.refetch} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Server rejected refresh");
  });

  it("classifies transport failure as unavailable and aborts replaced requests", async () => {
    const first = deferred<string[]>();
    const load = vi.fn((signal: AbortSignal) => {
      if (load.mock.calls.length === 1) return first.promise;
      return Promise.reject(new ApiError("UNAVAILABLE", "Offline", 0, "transport"));
    });
    const { result } = renderHook(() => useQuery("items", load, []));
    const firstSignal = load.mock.calls[0][0];
    act(() => result.current.refetch());
    expect(firstSignal.aborted).toBe(true);
    await waitFor(() => expect(result.current.unavailable).toBe(true));
    expect(result.current.error).toBeUndefined();
  });
});

describe("useMutation", () => {
  it("disables duplicate runs in flight and records success", async () => {
    const request = deferred<string>();
    const mutate = vi.fn(() => request.promise);
    const { result } = renderHook(() => useMutation(mutate));
    let running!: Promise<void>;
    act(() => { running = result.current.run("one"); });
    expect(result.current.pending).toBe(true);
    await act(async () => result.current.run("two"));
    expect(mutate).toHaveBeenCalledTimes(1);
    await act(async () => { request.resolve("done"); await running; });
    expect(result.current.result).toBe("done");
    expect(result.current.pending).toBe(false);
  });

  it("propagates ApiError details, resets, and blocks while offline", async () => {
    const failure = new ApiError("INVALID_ARGUMENT", "Criteria is required.", 400, "http");
    const { result, rerender } = renderHook(() => useMutation(async () => { throw failure; }));
    await act(async () => result.current.run(undefined));
    expect(result.current.error).toBe(failure);
    expect(result.current.error).toMatchObject({ code: "INVALID_ARGUMENT", message: "Criteria is required." });
    act(() => result.current.reset());
    expect(result.current.error).toBeUndefined();
    reachability.unreachable = true;
    rerender();
    expect(result.current.blocked).toBe(true);
    expect(result.current.disabledReason).toContain("Cannot reach the Gyrifi runtime");
    await act(async () => result.current.run(undefined));
    expect(result.current.error).toBeUndefined();
  });
});

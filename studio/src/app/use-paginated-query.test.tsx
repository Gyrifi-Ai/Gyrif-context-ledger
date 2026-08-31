import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ListPage } from "../api/types";
import { usePaginatedQuery } from "./use-paginated-query";

type Item = { id: string; label: string };

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((accept, fail) => { resolve = accept; reject = fail; });
  return { promise, resolve, reject };
}

describe("usePaginatedQuery", () => {
  it("merges pages without duplicates and exposes loading state", async () => {
    const second = deferred<ListPage<Item>>();
    const fetchPage = vi.fn((cursor: string | undefined) => cursor
      ? second.promise
      : Promise.resolve({ items: [{ id: "one", label: "One" }], nextCursor: "cursor-two" }));
    const { result } = renderHook(() => usePaginatedQuery("items", (cursor) => fetchPage(cursor), []));

    await waitFor(() => expect(result.current.data).toEqual([{ id: "one", label: "One" }]));
    act(() => result.current.loadMore());
    expect(result.current.loadingMore).toBe(true);
  expect(fetchPage).toHaveBeenLastCalledWith("cursor-two");
    await act(async () => second.resolve({ items: [{ id: "one", label: "duplicate" }, { id: "two", label: "Two" }] }));

    expect(result.current.data).toEqual([{ id: "one", label: "One" }, { id: "two", label: "Two" }]);
    expect(result.current.nextCursor).toBeUndefined();
    expect(result.current.loadingMore).toBe(false);
  });

  it("retains loaded rows when loading another page fails", async () => {
    const failure = new Error("Next page failed");
    const fetchPage = vi.fn((cursor: string | undefined) => cursor
      ? Promise.reject(failure)
      : Promise.resolve({ items: [{ id: "one", label: "One" }], nextCursor: "cursor-two" }));
    const { result } = renderHook(() => usePaginatedQuery("items", (cursor) => fetchPage(cursor), []));

    await waitFor(() => expect(result.current.nextCursor).toBe("cursor-two"));
    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.loadMoreError).toBe(failure));
    expect(result.current.data).toEqual([{ id: "one", label: "One" }]);
    expect(result.current.nextCursor).toBe("cursor-two");
  });
});

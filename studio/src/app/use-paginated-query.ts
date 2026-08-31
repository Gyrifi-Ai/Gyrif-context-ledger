import { useCallback, useEffect, useRef, useState } from "react";
import type { ListPage } from "../api/types";
import { useQuery, type QueryResult } from "./use-async";

export type PaginatedQueryResult<T> = Omit<QueryResult<T[]>, "data"> & {
  data: T[] | undefined;
  nextCursor: string | undefined;
  loadingMore: boolean;
  loadMoreError: Error | undefined;
  loadMore: () => void;
};

function asError(value: unknown): Error {
  return value instanceof Error ? value : new Error("The operation could not be completed.");
}

export function usePaginatedQuery<T extends { id: string }>(
  key: string,
  fetchPage: (cursor: string | undefined, signal: AbortSignal) => Promise<ListPage<T>>,
  deps: unknown[],
): PaginatedQueryResult<T> {
  const fetchRef = useRef(fetchPage);
  const moreControllerRef = useRef<AbortController | null>(null);
  const moreRequestRef = useRef(0);
  const mountedRef = useRef(true);
  const [data, setData] = useState<T[]>();
  const [nextCursor, setNextCursor] = useState<string>();
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState<Error>();
  fetchRef.current = fetchPage;

  const firstPage = useQuery(key, (signal) => fetchRef.current(undefined, signal), deps);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      moreControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    moreControllerRef.current?.abort();
    moreRequestRef.current++;
    setLoadingMore(false);
    setLoadMoreError(undefined);
    if (firstPage.data === undefined) {
      setData(undefined);
      setNextCursor(undefined);
      return;
    }
    setData(firstPage.data.items ?? []);
    setNextCursor(firstPage.data.nextCursor);
  }, [firstPage.data]);

  const loadMore = useCallback(() => {
    if (!nextCursor || loadingMore || firstPage.loading || firstPage.refetching) return;
    moreControllerRef.current?.abort();
    const controller = new AbortController();
    moreControllerRef.current = controller;
    const request = ++moreRequestRef.current;
    setLoadingMore(true);
    setLoadMoreError(undefined);
    void fetchRef.current(nextCursor, controller.signal)
      .then((page) => {
        if (!mountedRef.current || request !== moreRequestRef.current) return;
        setData((current) => {
          const seen = new Set((current ?? []).map((item) => item.id));
          return [...(current ?? []), ...(page.items ?? []).filter((item) => !seen.has(item.id))];
        });
        setNextCursor(page.nextCursor);
        setLoadingMore(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted || !mountedRef.current || request !== moreRequestRef.current) return;
        setLoadMoreError(asError(error));
        setLoadingMore(false);
      });
  }, [firstPage.loading, firstPage.refetching, loadingMore, nextCursor]);

  const visibleData = data ?? firstPage.data?.items;
  const visibleNextCursor = data === undefined ? firstPage.data?.nextCursor : nextCursor;

  return {
    data: visibleData,
    error: firstPage.error,
    loading: firstPage.loading,
    refetching: firstPage.refetching,
    unavailable: firstPage.unavailable,
    refetch: firstPage.refetch,
    nextCursor: visibleNextCursor,
    loadingMore,
    loadMoreError,
    loadMore,
  };
}

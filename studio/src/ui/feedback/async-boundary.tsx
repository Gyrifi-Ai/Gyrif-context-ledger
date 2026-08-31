import type { ReactNode } from "react";
import type { QueryResult } from "../../app/use-async";
import { ErrorState } from "./error-state";
import { Skeleton } from "./skeleton";

export function AsyncBoundary<T>({ query, empty, isEmpty = (data) => Array.isArray(data) && data.length === 0, children }: { query: QueryResult<T>; empty: ReactNode; isEmpty?: (data: T) => boolean; children: (data: T) => ReactNode }) {
  if (query.loading && query.data === undefined) return <Skeleton count={3} />;
  if (query.error) return <ErrorState message={query.error.message} onRetry={query.refetch} />;
  if (query.data === undefined) return <div className={query.unavailable ? "gy-is-refetching" : undefined}><Skeleton count={3} /></div>;
  if (isEmpty(query.data)) return <>{empty}</>;
  return <div className={query.refetching || query.unavailable ? "gy-is-refetching" : undefined}>{children(query.data)}</div>;
}
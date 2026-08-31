import { renderToStaticMarkup } from "react-dom/server";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import type { QueryResult } from "../../app/use-async";
import { AsyncBoundary } from "./async-boundary";

const refetch = vi.fn();

function query<T>(state: Partial<QueryResult<T>>): QueryResult<T> {
  return { data: undefined, error: undefined, loading: false, refetching: false, unavailable: false, refetch, ...state };
}

describe("AsyncBoundary", () => {
  it("renders loading, error, empty, and populated query states", () => {
    expect(renderToStaticMarkup(<AsyncBoundary query={query<string[]>({ loading: true })} empty="Empty">{(data) => data.join(",")}</AsyncBoundary>)).toContain("aria-busy=\"true\"");
    expect(renderToStaticMarkup(<AsyncBoundary query={query<string[]>({ error: new Error("Unavailable") })} empty="Empty">{(data) => data.join(",")}</AsyncBoundary>)).toContain("Unavailable");
    expect(renderToStaticMarkup(<AsyncBoundary query={query<string[]>({ data: [] })} empty="Empty">{(data) => data.join(",")}</AsyncBoundary>)).toContain("Empty");
    const html = renderToStaticMarkup(<AsyncBoundary query={query<string[]>({ data: ["ready"], refetching: true })} empty="Empty">{(data) => data.join(",")}</AsyncBoundary>);
    expect(html).toContain("ready");
    expect(html).toContain("gy-is-refetching");
  });

  it("passes retry through to the query refetch callback", () => {
    const boundary = AsyncBoundary({ query: query<string[]>({ error: new Error("Unavailable") }), empty: "Empty", children: (data) => data.join(",") }) as ReactElement<{ onRetry: () => void }>;
    boundary.props.onRetry();
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
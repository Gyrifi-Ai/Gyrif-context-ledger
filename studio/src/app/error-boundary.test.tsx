import { renderToStaticMarkup } from "react-dom/server";
import type { ErrorInfo, ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ErrorBoundary } from "./error-boundary";

afterEach(() => vi.restoreAllMocks());

describe("ErrorBoundary", () => {
  it("renders its fallback from captured error state and exposes reset", () => {
    const reset = vi.fn();
    const boundary = new ErrorBoundary({
      children: <p>Content</p>,
      fallback: (error, resetBoundary) => <button onClick={resetBoundary}>{error.message}</button>,
    });
    boundary.state = ErrorBoundary.getDerivedStateFromError(new Error("Render failed"));
    boundary.setState = reset as typeof boundary.setState;

    const html = renderToStaticMarkup(boundary.render() as ReactElement);
    boundary.reset();

    expect(html).toContain("Render failed");
    expect(reset).toHaveBeenCalledWith({ error: null });
  });

  it("logs and calls onError only once for the same captured error", () => {
    const log = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const onError = vi.fn();
    const boundary = new ErrorBoundary({ children: null, fallback: () => null, onError });
    const error = new Error("Render failed");
    const info = { componentStack: "\n at BrokenPage" } as ErrorInfo;

    boundary.componentDidCatch(error, info);
    boundary.componentDidCatch(error, info);

    expect(log).toHaveBeenCalledTimes(1);
    expect(log).toHaveBeenCalledWith(error, info.componentStack);
    expect(onError).toHaveBeenCalledTimes(1);
  });
});
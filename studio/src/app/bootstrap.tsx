import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ApiError, getLastFailedRequestId } from "../api/client";
import { CodeBlock } from "../ui/patterns/code-block";
import { ErrorState } from "../ui/feedback/error-state";
import { ErrorBoundary } from "./error-boundary";
import { Providers } from "./providers";
import { Shell } from "./shell";

function RootFallback(error: Error, reset: () => void) {
  const requestId = error instanceof ApiError ? error.requestId : getLastFailedRequestId();
  return (
    <main className="grid min-h-screen place-items-center bg-background p-6">
      <div className="w-full max-w-2xl space-y-4">
        <ErrorState title="Studio could not render" message="The application encountered an unexpected error. Reset Studio to try again." onRetry={reset} actionLabel="Reload Studio" />
        <CodeBlock value={error.message} language="error" />
        {requestId && <p className="font-mono text-xs text-muted-foreground">Request ID: {requestId}</p>}
      </div>
    </main>
  );
}

export function bootstrap(element: HTMLElement) {
  // ErrorBoundary is the single logging owner so caught render failures are not reported twice.
  createRoot(element, { onCaughtError: () => undefined }).render(<StrictMode><Providers><ErrorBoundary fallback={RootFallback}><Shell /></ErrorBoundary></Providers></StrictMode>);
}

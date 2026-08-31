import { Button } from "../ui/primitives/button";
import { runtimeUnavailableMessage, useReachability } from "./reachability";

export function ReachabilityBanner() {
  const { unreachable, retry } = useReachability();
  if (!unreachable) return null;
  return (
    <div role="alert" className="flex flex-wrap items-center justify-between gap-3 border-b border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive md:px-8">
      <span>{runtimeUnavailableMessage}</span>
      <Button variant="secondary" size="sm" onClick={retry}>Retry</Button>
    </div>
  );
}
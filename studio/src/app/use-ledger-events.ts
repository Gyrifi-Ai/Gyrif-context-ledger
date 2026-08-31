import { useEffect } from "react";
import { useReachability } from "./reachability";

export function useLedgerEvents(onInvalidate: () => void): void {
  const { registerInvalidation } = useReachability();
  useEffect(() => {
    return registerInvalidation(onInvalidate);
  }, [onInvalidate, registerInvalidation]);
}
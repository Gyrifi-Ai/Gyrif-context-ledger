import { useEffect } from "react";
import { useReachability } from "./reachability";

export function useLedgerEvents(ledgerId: string | undefined, onInvalidate: () => void): void {
  const { registerInvalidation } = useReachability();
  useEffect(() => {
    return registerInvalidation(ledgerId, onInvalidate);
  }, [ledgerId, onInvalidate, registerInvalidation]);
}
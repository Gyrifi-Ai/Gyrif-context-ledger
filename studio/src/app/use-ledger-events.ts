import { useEffect } from "react";
import { subscribeToEvents } from "../api/events";

// GRF-210 replaces the handshake-only stream with domain events.
const ledgerEventsEnabled = import.meta.env.VITE_GYRIFI_ENABLE_LEDGER_EVENTS === "true";

export function useLedgerEvents(onInvalidate: () => void): void {
  useEffect(() => {
    if (!ledgerEventsEnabled) return;
    return subscribeToEvents(onInvalidate);
  }, [onInvalidate]);
}
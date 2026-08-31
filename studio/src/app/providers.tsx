import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { api } from "../api/client";
import type { Ledger } from "../api/types";
import { ReachabilityProvider } from "./reachability";
import { useQuery } from "./use-async";

interface AppState {
  ledgerId: string;
  ledger: Ledger | null;
  ledgers: Ledger[];
  setLedgerId: (id: string) => void;
  refreshLedgers: () => Promise<void>;
  openLedgerSwitcher: () => void;
  ledgerSwitcherRequest: number;
}

const Context = createContext<AppState | null>(null);

function AppStateProvider({ children }: { children: ReactNode }) {
  const [ledgerId, setLedgerId] = useState(() => localStorage.getItem("gyrifi.ledger") ?? "");
  const [ledgerSwitcherRequest, setLedgerSwitcherRequest] = useState(0);
  const ledgerQuery = useQuery("app-ledgers", async (signal) => (await api.ledgers({ signal })).items ?? [], []);
  const ledgers = ledgerQuery.data ?? [];
  const refreshLedgers = useCallback(async () => { ledgerQuery.refetch(); }, [ledgerQuery.refetch]);
  const selectLedger = useCallback((id: string) => { localStorage.setItem("gyrifi.ledger", id); setLedgerId(id); }, []);
  const openLedgerSwitcher = useCallback(() => setLedgerSwitcherRequest((request) => request + 1), []);
  const ledger = ledgers.find((item) => item.id === ledgerId) ?? null;
  const value = useMemo(() => ({ ledgerId, ledger, ledgers, setLedgerId: selectLedger, refreshLedgers, openLedgerSwitcher, ledgerSwitcherRequest }), [ledgerId, ledger, ledgers, selectLedger, refreshLedgers, openLedgerSwitcher, ledgerSwitcherRequest]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function Providers({ children }: { children: ReactNode }) {
  return <ReachabilityProvider><AppStateProvider>{children}</AppStateProvider></ReachabilityProvider>;
}

export function useAppState() {
  const value = useContext(Context);
  if (!value) throw new Error("App providers are missing");
  return value;
}

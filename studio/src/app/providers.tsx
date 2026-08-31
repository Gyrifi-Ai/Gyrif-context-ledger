import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api } from "../api/client";
import type { Ledger } from "../api/types";

interface AppState {
  ledgerId: string;
  ledger: Ledger | null;
  ledgers: Ledger[];
  setLedgerId: (id: string) => void;
  refreshLedgers: () => Promise<void>;
}

const Context = createContext<AppState | null>(null);

export function Providers({ children }: { children: ReactNode }) {
  const [ledgerId, setLedgerId] = useState(() => localStorage.getItem("gyrifi.ledger") ?? "");
  const [ledgers, setLedgers] = useState<Ledger[]>([]);
  const refreshLedgers = useCallback(async () => { const result = await api.ledgers(); setLedgers(result.items ?? []); }, []);
  useEffect(() => { void refreshLedgers().catch(() => undefined); }, [refreshLedgers]);
  const selectLedger = useCallback((id: string) => { localStorage.setItem("gyrifi.ledger", id); setLedgerId(id); }, []);
  const ledger = ledgers.find((item) => item.id === ledgerId) ?? null;
  const value = useMemo(() => ({ ledgerId, ledger, ledgers, setLedgerId: selectLedger, refreshLedgers }), [ledgerId, ledger, ledgers, selectLedger, refreshLedgers]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useAppState() {
  const value = useContext(Context);
  if (!value) throw new Error("App providers are missing");
  return value;
}

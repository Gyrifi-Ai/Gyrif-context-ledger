import { createContext, useContext, useMemo, useState, type ReactNode } from "react";

interface AppState {
  ledgerId: string;
  setLedgerId: (id: string) => void;
}

const Context = createContext<AppState | null>(null);

export function Providers({ children }: { children: ReactNode }) {
  const [ledgerId, setLedgerId] = useState(() => localStorage.getItem("gyrifi.ledger") ?? "");
  const value = useMemo(() => ({ ledgerId, setLedgerId: (id: string) => { localStorage.setItem("gyrifi.ledger", id); setLedgerId(id); } }), [ledgerId]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useAppState() {
  const value = useContext(Context);
  if (!value) throw new Error("App providers are missing");
  return value;
}

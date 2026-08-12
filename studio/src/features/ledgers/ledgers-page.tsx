import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Ledger } from "../../api/types";
import { useAppState } from "../../app/providers";
import { Button } from "../../ui/primitives/button";
import { EmptyState } from "../../ui/feedback/empty-state";

export function LedgersPage() {
  const [items, setItems] = useState<Ledger[]>([]);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const { ledgerId, setLedgerId } = useAppState();
  const load = () => api.ledgers().then((value) => setItems(value.items)).catch((value: Error) => setError(value.message));
  useEffect(() => { void load(); }, []);
  const create = async (event: FormEvent) => {
    event.preventDefault(); setError("");
    try { const ledger = await api.createLedger({ name }); setName(""); setLedgerId(ledger.id); await load(); } catch (value) { setError((value as Error).message); }
  };
  return <section className="content-grid">
    <div className="panel panel--wide"><div className="panel__heading"><div><span className="eyebrow">GOVERNED CONTEXT</span><h2>Your ledgers</h2></div><span>{items.length} total</span></div>
      {items.length === 0 ? <EmptyState title="Create the first ledger">A ledger governs one Qdrant context collection.</EmptyState> : <div className="cards">{items.map((ledger) => <button className={`ledger-card ${ledger.id === ledgerId ? "selected" : ""}`} onClick={() => setLedgerId(ledger.id)} key={ledger.id}><span className="ledger-card__icon">L</span><span><b>{ledger.name}</b><small>{ledger.description || "Qdrant context ledger"}</small></span><code>{ledger.id.slice(0, 12)}</code></button>)}</div>}
    </div>
    <form className="panel" onSubmit={create}><span className="eyebrow">NEW LEDGER</span><h2>Create ledger</h2><label>Name<input value={name} onChange={(event) => setName(event.target.value)} required /></label>{error && <p className="error">{error}</p>}<Button type="submit">Create ledger</Button></form>
  </section>;
}

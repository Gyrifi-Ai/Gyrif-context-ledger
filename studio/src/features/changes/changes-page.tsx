import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Change } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { Button } from "../../ui/primitives/button";
import { StatusBadge } from "../../ui/patterns/status-badge";

export function ChangesPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Change[]>([]);
  const [unit, setUnit] = useState("");
  const [desired, setDesired] = useState("{}");
  const [error, setError] = useState("");
  const load = () => { if (ledgerId) void api.changes(ledgerId).then((value) => setItems(value.items)).catch((value: Error) => setError(value.message)); };
  useEffect(() => { load(); }, [ledgerId]);
  const create = async (event: FormEvent) => {
    event.preventDefault(); setError("");
    try { await api.createChange(ledgerId, { unit, action: "PUT", desired: JSON.parse(desired), idempotencyKey: crypto.randomUUID() }); setUnit(""); await load(); } catch (value) { setError((value as Error).message); }
  };
  if (!ledgerId) return <EmptyState title="Select a ledger">Choose or create a ledger before submitting Changes.</EmptyState>;
  return <section className="content-grid"><div className="panel panel--wide"><div className="panel__heading"><div><span className="eyebrow">DURABLE INBOX</span><h2>Proposed changes</h2></div><span>{items.length} received</span></div>
    {items.length === 0 ? <EmptyState title="No Changes yet">Submit desired state here or through the versioned API.</EmptyState> : <div className="table">{items.map((change) => <div className="table__row" key={change.id}><code>{change.id.slice(0, 12)}</code><b>{change.unit}</b><span>{change.action}</span><StatusBadge value={change.status} /></div>)}</div>}
  </div><form className="panel" onSubmit={create}><span className="eyebrow">INGEST</span><h2>Submit desired state</h2><label>Logical unit<input value={unit} onChange={(event) => setUnit(event.target.value)} placeholder="point/42" required /></label><label>JSON value<textarea value={desired} onChange={(event) => setDesired(event.target.value)} rows={8} /></label>{error && <p className="error">{error}</p>}<Button>Accept Change</Button></form></section>;
}

import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Change, Proposal } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { Button } from "../../ui/primitives/button";
import { StatusBadge } from "../../ui/patterns/status-badge";

export function ProposalsPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Proposal[]>([]);
  const [changes, setChanges] = useState<Change[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [title, setTitle] = useState("");
  const [error, setError] = useState("");
  const load = async () => { if (!ledgerId) return; try { const [proposalResult, changeResult] = await Promise.all([api.proposals(ledgerId), api.changes(ledgerId)]); setItems(proposalResult.items); setChanges(changeResult.items.filter((change) => change.status === "READY")); } catch (value) { setError((value as Error).message); } };
  useEffect(() => { void load(); }, [ledgerId]);
  const create = async (event: FormEvent) => { event.preventDefault(); try { await api.createProposal(ledgerId, { title, changeIds: selected }); setTitle(""); setSelected([]); await load(); } catch (value) { setError((value as Error).message); } };
  const act = async (action: "evaluate" | "approve" | "release", proposal: Proposal) => { try { if (action === "evaluate") await api.evaluate(ledgerId, proposal.id, "The proposed context is accurate, internally consistent, and safe to release."); if (action === "approve") await api.approve(ledgerId, proposal.id); if (action === "release") await api.release(ledgerId, proposal.id); await load(); } catch (value) { setError((value as Error).message); } };
  if (!ledgerId) return <EmptyState title="Select a ledger">Proposals are scoped to a ledger.</EmptyState>;
  return <section className="content-grid"><div className="panel panel--wide"><div className="panel__heading"><div><span className="eyebrow">CONTEXT PRS</span><h2>Review queue</h2></div></div>{items.length === 0 ? <EmptyState title="No Proposals">Select Ready Changes to create a reviewable Context PR.</EmptyState> : <div className="proposal-list">{items.map((proposal) => <article key={proposal.id}><div><StatusBadge value={proposal.status} /><h3>{proposal.title}</h3><code>{proposal.hash.slice(0, 18)} · {proposal.changeIds.length} changes</code></div><div className="actions"><Button onClick={() => void act("evaluate", proposal)}>Evaluate</Button><Button onClick={() => void act("approve", proposal)}>Approve</Button><Button onClick={() => void act("release", proposal)}>Release</Button></div></article>)}</div>}</div>
  <form className="panel" onSubmit={create}><span className="eyebrow">NEW CONTEXT PR</span><h2>Select Changes</h2><label>Title<input value={title} onChange={(event) => setTitle(event.target.value)} required /></label><div className="checklist">{changes.map((change) => <label key={change.id}><input type="checkbox" checked={selected.includes(change.id)} onChange={() => setSelected((old) => old.includes(change.id) ? old.filter((id) => id !== change.id) : [...old, change.id])} /> {change.unit}</label>)}</div>{error && <p className="error">{error}</p>}<Button disabled={selected.length === 0}>Create Proposal</Button></form></section>;
}

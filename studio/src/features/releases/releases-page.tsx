import { useEffect, useState } from "react";
import { api } from "../../api/client";
import type { Release } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { Button } from "../../ui/primitives/button";

export function ReleasesPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Release[]>([]);
  const [error, setError] = useState("");
  useEffect(() => { if (ledgerId) api.releases(ledgerId).then((value) => setItems(value.items)).catch((value: Error) => setError(value.message)); }, [ledgerId]);
  const rollback = async (releaseId: string) => { try { await api.rollback(ledgerId, releaseId); window.location.hash = "proposals"; } catch (value) { setError((value as Error).message); } };
  if (!ledgerId) return <EmptyState title="Select a ledger">Release history is scoped to a ledger.</EmptyState>;
  return <section className="panel"><div className="panel__heading"><div><span className="eyebrow">IMMUTABLE HISTORY</span><h2>Releases</h2></div><span>{items.length} releases</span></div>{error && <p className="error">{error}</p>}{items.length === 0 ? <EmptyState title="No Releases yet">Approved Proposals become immutable Releases here.</EmptyState> : <div className="timeline">{items.map((release, index) => <article key={release.id}><span className="timeline__node" /><div><small>{index === 0 ? "HEAD" : "HISTORY"}</small><h3>{release.id}</h3><code>{release.hash}</code><p>Proposal {release.proposalId}</p>{index > 0 && <Button onClick={() => void rollback(release.id)}>Create rollback Proposal</Button>}</div></article>)}</div>}</section>;
}

import type { ReactNode } from "react";
import type { Route } from "../../app/router";

const areas: { id: Route; label: string }[] = [
  { id: "ledgers", label: "Ledgers" },
  { id: "changes", label: "Changes" },
  { id: "proposals", label: "Proposals" },
  { id: "releases", label: "Releases" },
];

export function ApplicationShell({ route, children, ledgerId }: { route: Route; children: ReactNode; ledgerId: string }) {
  return <div className="shell">
    <aside>
      <div className="brand"><span className="brand__mark">G</span><div><b>Gyrifi</b><small>Context ledger</small></div></div>
      <nav>{areas.map((area) => <a className={route === area.id ? "active" : ""} href={`#${area.id}`} key={area.id}>{area.label}</a>)}</nav>
      <div className="aside__footer"><span className="health-dot" /> Runtime connected</div>
    </aside>
    <main><header><div><span className="eyebrow">MODULAR MONOLITH</span><h1>{areas.find((area) => area.id === route)?.label}</h1></div><code>{ledgerId || "No ledger selected"}</code></header>{children}</main>
  </div>;
}

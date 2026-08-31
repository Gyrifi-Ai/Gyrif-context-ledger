import { useRoute } from "./router";
import { useAppState } from "./providers";
import { ErrorBoundary } from "./error-boundary";
import { ReachabilityBanner } from "./reachability-banner";
import { ApplicationShell } from "../ui/layout/application-shell";
import { LedgersPage } from "../features/ledgers/ledgers-page";
import { ChangesPage } from "../features/changes/changes-page";
import { ProposalsPage } from "../features/proposals/proposals-page";
import { ReleasesPage } from "../features/releases/releases-page";
import { Nav } from "../features/shell/nav";
import { LedgerSwitcher } from "../features/shell/ledger-switcher";
import { HeadChip } from "../features/shell/head-chip";
import { RuntimeStatus } from "../features/shell/runtime-status";
import { PageHeader } from "../ui/layout/page-header";
import { ErrorState } from "../ui/feedback/error-state";
import { CodeBlock } from "../ui/patterns/code-block";

const pageInfo = {
  ledgers: { eyebrow: "Ledgers", title: "Ledgers", description: "A ledger is a governed namespace with its own inbox, proposals, and release history." },
  changes: { eyebrow: "Durable inbox", title: "Changes", description: "Desired-state mutations waiting to be proposed. Nothing here has touched the target." },
  proposals: { eyebrow: "Context PRs", title: "Proposals", description: "Review batched changes, attach evidence, approve, and release." },
  releases: { eyebrow: "Immutable history", title: "Releases", description: "Every release was applied to the target and verified before it was recorded." },
};

export function Shell() {
  const route = useRoute();
  const { ledgerId } = useAppState();
  const pages = { ledgers: <LedgersPage />, changes: <ChangesPage />, proposals: <ProposalsPage />, releases: <ReleasesPage /> };
  return <ApplicationShell sidebar={<div className="flex h-full flex-row items-center gap-4 p-3 md:flex-col md:items-stretch md:gap-6 md:p-4"><a href="#ledgers" className="flex items-center gap-3 px-2"><span className="grid size-9 place-items-center rounded-md bg-primary font-semibold text-primary-foreground">G</span><span className="hidden md:block"><b className="block text-sm">Gyrifi</b><small className="text-[11px] uppercase tracking-wider text-muted-foreground">Context ledger</small></span></a><Nav route={route} ledgerId={ledgerId} /><div className="ml-auto flex items-center gap-3 md:ml-0 md:mt-auto md:block"><div className="hidden px-2 pb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground md:block">Active ledger</div><div className="overflow-visible rounded-md border border-border bg-card shadow-sm"><LedgerSwitcher /><div className="border-t border-border/60 bg-muted/40 px-3 py-2"><RuntimeStatus /></div></div></div></div>} topbar={<HeadChip ledgerId={ledgerId} />} banner={<ReachabilityBanner />} header={<PageHeader {...pageInfo[route]} />}><ErrorBoundary key={route} fallback={(error, reset) => <div className="space-y-4"><ErrorState title="This page could not render" message="The rest of Studio is still available. Reset this page or choose another area." onRetry={reset} actionLabel="Reset page" /><CodeBlock value={error.message} language="error" /></div>}>{pages[route]}</ErrorBoundary></ApplicationShell>;
}

import { useRoute } from "./router";
import { useAppState } from "./providers";
import { ApplicationShell } from "../ui/layout/application-shell";
import { LedgersPage } from "../features/ledgers/ledgers-page";
import { ChangesPage } from "../features/changes/changes-page";
import { ProposalsPage } from "../features/proposals/proposals-page";
import { ReleasesPage } from "../features/releases/releases-page";

export function Shell() {
  const route = useRoute();
  const { ledgerId } = useAppState();
  const pages = { ledgers: <LedgersPage />, changes: <ChangesPage />, proposals: <ProposalsPage />, releases: <ReleasesPage /> };
  return <ApplicationShell route={route} ledgerId={ledgerId}>{pages[route]}</ApplicationShell>;
}

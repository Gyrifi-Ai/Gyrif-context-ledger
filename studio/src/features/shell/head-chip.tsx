import { api } from "../../api/client";
import { useQuery } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { HashChip } from "../../ui/patterns/hash-chip";

export function HeadChip({ ledgerId }: { ledgerId: string }) {
	const releasesQuery = useQuery("head", async (signal) => ledgerId ? (await api.releases(ledgerId, { signal })).items ?? [] : [], [ledgerId]);
	if (!ledgerId) return <span className="text-xs text-muted-foreground">No releases yet</span>;
	if (releasesQuery.error) return <ErrorState title="Unable to load HEAD" message={releasesQuery.error.message} onRetry={releasesQuery.refetch} />;
	const head = releasesQuery.data?.[0]?.id;
	return head ? <HashChip label="HEAD" value={head} /> : <span className="text-xs text-muted-foreground">No releases yet</span>;
}

import { useEffect, useState } from "react";
import { api } from "../../api/client";
import { HashChip } from "../../ui/patterns/hash-chip";
export function HeadChip({ ledgerId }: { ledgerId: string }) { const [head, setHead] = useState(""); useEffect(() => { if (!ledgerId) { setHead(""); return; } void api.releases(ledgerId).then((value) => setHead(value.items?.[0]?.id ?? "")).catch(() => setHead("")); }, [ledgerId]); return head ? <HashChip label="HEAD" value={head} /> : <span className="text-xs text-muted-foreground">No releases yet</span>; }

import type { Change } from "../../api/types";
import { Drawer } from "../../ui/layout/drawer";
import { CodeBlock } from "../../ui/patterns/code-block";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { changeTone } from "../shared/status";

export function ChangeDetailDrawer({ change, onClose }: { change: Change | null; onClose: () => void }) {
  return (
    <Drawer open={change !== null} onClose={onClose} title="Change detail">
      {change && (
        <div className="grid gap-5 text-sm">
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-3">
            <dt className="text-muted-foreground">ID</dt><dd className="break-all font-mono text-xs">{change.id}</dd>
            <dt className="text-muted-foreground">Sequence</dt><dd>{change.sequence}</dd>
            <dt className="text-muted-foreground">Unit</dt><dd className="break-all font-mono text-xs">{change.unit}</dd>
            <dt className="text-muted-foreground">Action</dt><dd>{change.action}</dd>
            <dt className="text-muted-foreground">Status</dt><dd><StatusBadge label={change.status} tone={changeTone(change.status)} /></dd>
            <dt className="text-muted-foreground">Created</dt><dd><time dateTime={change.createdAt}>{change.createdAt}</time></dd>
          </dl>
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Base fingerprint</p>{change.baseFingerprint ? <HashChip value={change.baseFingerprint} /> : <p>Not captured — base fingerprints are recorded from GRF-221 onward.</p>}</div>
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Desired fingerprint</p><HashChip value={change.desiredFingerprint} /></div>
          <div><p className="mb-2 text-xs font-medium text-muted-foreground">Desired value</p><CodeBlock value={change.desired ?? null} /></div>
        </div>
      )}
    </Drawer>
  );
}
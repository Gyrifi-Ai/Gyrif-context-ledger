import type { Release, ReleaseIntentOperation } from "../../api/types";
import { Drawer } from "../../ui/layout/drawer";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { cn } from "@/lib/utils";

function Fingerprint({ label, value }: { label: string; value?: string }) {
  return <div><dt className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</dt><dd className="mt-1 break-all font-mono text-xs">{value || "Not present"}</dd></div>;
}

export function PlanOperations({ operations }: { operations: ReleaseIntentOperation[] }) {
  return (
    <ol className="grid gap-3">
      {operations.map((operation, index) => {
        const missingBeforeImage = operation.beforeExists && !operation.hasBeforeImage;
        return (
          <li key={`${operation.unit}-${index}`} className={cn("rounded-md border border-border bg-background/50 p-3", missingBeforeImage && "border-warning/50 bg-warning/10")}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <code className="break-all text-sm font-medium">{operation.unit}</code>
              <StatusBadge label={operation.action} tone={operation.action === "DELETE" ? "danger" : "info"} />
            </div>
            <dl className="mt-3 grid gap-3 sm:grid-cols-2">
              <Fingerprint label="Expected fingerprint" value={operation.expectedFingerprint} />
              <Fingerprint label="Desired fingerprint" value={operation.desiredFingerprint} />
            </dl>
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
              {missingBeforeImage ? <strong className="text-warning">No rollback material for this unit.</strong> : operation.beforeExists ? <span className="text-success">Before-image retained</span> : <span className="text-muted-foreground">No prior value; rollback will delete this unit.</span>}
              {operation.targetMetric && <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-muted-foreground">Target metric: {operation.targetMetric}</span>}
            </div>
          </li>
        );
      })}
    </ol>
  );
}

export function PlanDrawer({ open, onClose, release, operations = [] }: { open: boolean; onClose: () => void; release?: Release; operations?: ReleaseIntentOperation[] }) {
  const metrics = [...new Set(operations.map((operation) => operation.targetMetric).filter((metric): metric is string => Boolean(metric)))];
  return (
    <Drawer open={open} onClose={onClose} title="Release plan">
      {release && <div className="mb-4 flex flex-wrap items-center gap-2"><HashChip value={release.id} label="Release" />{metrics.map((metric) => <span key={metric} className="text-xs text-muted-foreground">Target metric: <strong className="font-mono text-foreground">{metric}</strong></span>)}</div>}
      <p className="mb-4 text-sm text-muted-foreground">Compiled operations and retained rollback material recorded before target mutation.</p>
      <PlanOperations operations={operations} />
    </Drawer>
  );
}
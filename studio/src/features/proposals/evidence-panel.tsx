import { useEffect, useMemo, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { CheckResult, EvaluationResponse, Proposal } from "../../api/types";
import { useSystemStatus } from "../../app/reachability";
import { useMutation } from "../../app/use-async";
import { ErrorState } from "../../ui/feedback/error-state";
import { Skeleton } from "../../ui/feedback/skeleton";
import { Button } from "../../ui/primitives/button";
import { Field } from "../../ui/primitives/field";
import { Textarea } from "../../ui/primitives/textarea";
import { HashChip } from "../../ui/patterns/hash-chip";
import { StatusBadge, type StatusTone } from "../../ui/patterns/status-badge";
import { criteriaPresets } from "./criteria-presets";
import { evidenceView } from "./proposal-view";

function storedCriteria(proposalId: string): string {
  return typeof localStorage === "undefined" || typeof localStorage.getItem !== "function" ? "" : localStorage.getItem(`gyrifi.criteria.${proposalId}`) ?? "";
}

function findingTone(severity: string): StatusTone {
  switch (severity.toLowerCase()) {
    case "error":
    case "critical": return "danger";
    case "warning": return "warning";
    case "info": return "info";
    default: return "neutral";
  }
}

export function EvidencePanel({ ledgerId, proposal, check, onRefresh }: { ledgerId: string; proposal: Proposal; check?: CheckResult; onRefresh: () => void }) {
  const health = useSystemStatus();
  const [criteria, setCriteria] = useState(() => storedCriteria(proposal.id));
  const evaluation = useMutation(async (value: string) => {
    const response = await api.evaluate(ledgerId, proposal.id, value);
    onRefresh();
    return response;
  });

  useEffect(() => {
    setCriteria(storedCriteria(proposal.id));
    evaluation.reset();
  }, [proposal.id]);

  const setAndPersist = (value: string) => {
    setCriteria(value);
    if (typeof localStorage !== "undefined" && typeof localStorage.setItem === "function") localStorage.setItem(`gyrifi.criteria.${proposal.id}`, value);
    evaluation.reset();
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (criteria.trim()) void evaluation.run(criteria.trim());
  };
  const currentResult = evaluation.result as EvaluationResponse | undefined;
  const view = useMemo(() => evidenceView(check, currentResult), [check, currentResult]);
  const passed = currentResult?.passed ?? check?.passed;
  const summary = currentResult?.summary ?? check?.summary;

  return (
    <div className="grid gap-4">
      {check && !check.current && <div role="alert" className="rounded-md border border-warning/40 bg-warning/10 p-3 text-sm text-warning">Evidence was recorded for a different proposal hash and no longer applies.</div>}
  <form className="grid gap-3" onSubmit={submit}>
        <Field label="Evaluation criteria" hint="Saved locally for this Proposal.">
          <Textarea value={criteria} onChange={(event) => setAndPersist(event.target.value)} rows={5} required placeholder="Describe what must be true before this context can be approved." />
        </Field>
        <div className="flex flex-wrap gap-2" aria-label="Criteria presets">
          {criteriaPresets.map((preset) => <Button key={preset} variant="secondary" size="sm" onClick={() => setAndPersist(preset)}>{preset}</Button>)}
        </div>
        <div><Button type="submit" loading={evaluation.pending} disabled={!criteria.trim() || evaluation.blocked} title={evaluation.disabledReason}>Run evaluation</Button></div>
      </form>
      {evaluation.pending && <article className="grid gap-3 rounded-md border border-border bg-muted/40 p-4" aria-label="Running evaluation"><Skeleton height="1.25rem" width="30%" /><Skeleton height="3rem" /><Skeleton height="4rem" /></article>}
      {evaluation.error && <ErrorState title="Unable to evaluate Proposal" message={evaluation.error.message} onRetry={() => void evaluation.run(criteria.trim())} retryDisabled={evaluation.blocked || !criteria.trim()} retryTitle={evaluation.disabledReason} />}
      {(check || currentResult) ? (
        <article className="grid gap-4 rounded-md border border-border bg-muted/40 p-4" aria-label="Evaluation evidence">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge label={passed ? "PASS" : "FAIL"} tone={passed ? "success" : "danger"} />
            {check && <StatusBadge label={check.kind === "deterministic" ? "DETERMINISTIC" : "NATURAL LANGUAGE"} tone="info" />}
          </div>
          <p className="text-sm">{summary}</p>
          {check?.evidenceUnavailable && <p role="alert" className="text-sm text-destructive">Stored evidence is unavailable.</p>}
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
            <dt className="text-muted-foreground">Preview fidelity</dt><dd className="font-mono text-xs">{view.previewFidelity ?? "Not retained"}</dd>
            <dt className="text-muted-foreground">Model</dt><dd className="font-mono text-xs">{view.model ?? (check?.kind === "deterministic" ? "Deterministic checks" : "Not reported")}</dd>
            {check && <><dt className="text-muted-foreground">Bound hash</dt><dd><HashChip value={check.proposalHash} /></dd></>}
          </dl>
          {view.previewFidelity === "FAST" && <p className="rounded-md border border-warning/40 bg-warning/10 p-3 text-sm text-warning">Preview is an overlay summary, not a simulated query result.</p>}
          {health.inference === "disabled" && <p className="text-sm text-muted-foreground">{check?.kind === "natural-language" ? "Natural-language evaluation is currently off. This stored check was recorded when inference was enabled." : "This check was deterministic. Natural-language evaluation is off."}</p>}
          <div>
            <h4 className="text-sm font-semibold">Findings</h4>
            {view.findings.length === 0 ? <p className="mt-2 text-sm text-muted-foreground">No findings recorded.</p> : (
              <ul className="mt-2 divide-y divide-border rounded-md border border-border bg-card">
                {view.findings.map((finding, index) => (
                  <li key={`${finding.unit ?? "proposal"}-${index}`} className="grid gap-1 p-3 sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-3">
                    <StatusBadge label={finding.severity.toUpperCase()} tone={findingTone(finding.severity)} />
                    <p className="text-sm"><span className="font-mono text-xs">{finding.unit ?? "proposal"}</span><span className="mx-2 text-muted-foreground">—</span>{finding.message}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </article>
      ) : <p className="text-sm text-muted-foreground">No evaluation evidence has been recorded.</p>}
    </div>
  );
}

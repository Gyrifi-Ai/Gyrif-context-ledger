import { Button } from "../../ui/primitives/button";

export function SelectionActionBar({ count, onClear, onCreateProposal }: { count: number; onClear: () => void; onCreateProposal: () => void }) {
  if (count === 0) return null;
  return (
    <div className="sticky bottom-4 z-10 mx-auto mt-4 flex max-w-xl items-center justify-between gap-4 rounded-lg border border-border-accent bg-card px-4 py-3 shadow-lg" role="region" aria-label="Selected Changes">
      <span className="text-sm font-medium">{count} selected</span>
      <span className="flex gap-2">
        <Button variant="ghost" size="sm" onClick={onClear}>Clear</Button>
        <Button size="sm" onClick={onCreateProposal}>Create proposal</Button>
      </span>
    </div>
  );
}
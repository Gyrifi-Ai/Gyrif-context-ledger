import { useEffect, useMemo, useRef, useState } from "react";
import { useAppState } from "../../app/providers";
import { cn } from "@/lib/utils";

export function LedgerSwitcher() {
  const { ledger, ledgers, ledgerSwitcherRequest, setLedgerId, loadMoreLedgers, hasMoreLedgers, loadingMoreLedgers } = useAppState();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const items = useMemo(() => ledgers.filter((item) => item.name.toLowerCase().includes(query.toLowerCase())), [ledgers, query]);

  useEffect(() => {
    if (ledgerSwitcherRequest === 0) return;
    setOpen(true);
    trigger.current?.focus();
  }, [ledgerSwitcherRequest]);

  useEffect(() => {
    if (!open) return;
    const outside = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, [open]);

  const select = (index: number) => {
    const item = items[index];
    if (!item) return;
    setLedgerId(item.id);
    setOpen(false);
    setQuery("");
  };

  return (
    <div ref={root} className="relative w-full">
      <button ref={trigger} type="button" aria-haspopup="listbox" aria-expanded={open} onClick={() => setOpen(!open)} className="flex w-full items-center justify-between px-3 py-2.5 text-sm font-medium hover:bg-muted">
        <span className="flex min-w-0 items-center gap-2">
          <span className={cn("size-2 rounded-full", ledger ? "bg-success" : "bg-muted-foreground/40")} />
          <span className="truncate">{ledger?.name ?? "Select ledger"}</span>
        </span>
        <span aria-hidden="true">⌄</span>
      </button>
      {open && (
        <div role="listbox" className="absolute bottom-full left-0 z-10 mb-2 w-full rounded-md border border-border bg-card p-2 shadow-lg">
          <input
            autoFocus
            value={query}
            onChange={(event) => { setQuery(event.target.value); setActive(0); }}
            placeholder="Filter ledgers"
            className="mb-2 h-8 w-full rounded-sm border border-input bg-muted px-2 text-xs"
            onKeyDown={(event) => {
              if (event.key === "Escape") setOpen(false);
              if (event.key === "ArrowDown") { event.preventDefault(); setActive((value) => Math.min(value + 1, items.length - 1)); }
              if (event.key === "ArrowUp") { event.preventDefault(); setActive((value) => Math.max(value - 1, 0)); }
              if (event.key === "Enter") { event.preventDefault(); select(active); }
            }}
          />
          {items.length === 0 && <p className="px-2 py-3 text-xs text-muted-foreground">No ledgers match.</p>}
          {items.map((item, index) => (
            <button key={item.id} type="button" role="option" aria-selected={item.id === ledger?.id} onMouseEnter={() => setActive(index)} onClick={() => select(index)} className={cn("block w-full rounded-sm px-2 py-2 text-left text-xs", index === active ? "bg-muted" : "hover:bg-muted/60")}>
              {item.name}
            </button>
          ))}
          {hasMoreLedgers && !query && <button type="button" disabled={loadingMoreLedgers} onClick={loadMoreLedgers} className="mt-2 block w-full rounded-sm px-2 py-2 text-left text-xs font-medium text-primary hover:bg-muted disabled:opacity-60">{loadingMoreLedgers ? "Loading…" : "Load more"}</button>}
        </div>
      )}
    </div>
  );
}

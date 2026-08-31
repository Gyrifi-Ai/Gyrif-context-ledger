import { cn } from "@/lib/utils";
export function Segmented({ value, onChange, options }: { value: string; onChange: (value: string) => void; options: { value: string; label: string }[] }) {
  return <div className="inline-flex rounded-sm border border-border bg-muted p-1" role="group">{options.map((option) => <button key={option.value} type="button" onClick={() => onChange(option.value)} className={cn("h-7 rounded-sm px-3 text-xs font-medium", value === option.value ? "bg-card text-foreground shadow-sm" : "text-muted-foreground")}>{option.label}</button>)}</div>;
}

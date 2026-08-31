import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ProgressStep } from "./proposal-view";

export function ProgressRail({ steps }: { steps: ProgressStep[] }) {
  return (
    <ol aria-label="Proposal progress" className="grid grid-cols-4 gap-2 border-y border-border py-4">
      {steps.map((step, index) => (
        <li key={step.label} className="relative grid justify-items-center gap-2 text-center">
          {index > 0 && <span aria-hidden="true" className={cn("absolute right-1/2 top-3 h-px w-full", step.state === "pending" ? "bg-border" : "bg-primary")} />}
          <span className={cn(
            "relative z-10 grid size-6 place-items-center rounded-full border bg-card text-[11px] font-semibold",
            step.state === "complete" && "border-primary bg-primary text-primary-foreground",
            step.state === "current" && "border-2 border-primary text-primary",
            step.state === "pending" && "border-border text-muted-foreground",
          )}>
            {step.state === "complete" ? <Check className="size-3.5" aria-hidden="true" /> : index + 1}
          </span>
          <span className={cn("text-xs font-medium", step.state === "pending" ? "text-muted-foreground" : "text-foreground")}>{step.label}</span>
        </li>
      ))}
    </ol>
  );
}

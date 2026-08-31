import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import type { StatusTone } from "./status-badge";
export type TimelineItem = { id: string; node?: ReactNode; title: ReactNode; meta?: ReactNode; body?: ReactNode; tone?: StatusTone; current?: boolean };
export function Timeline({ items }: { items: TimelineItem[] }) { return <ol className="ml-2 border-l border-border">{items.map((item) => <li key={item.id} className="relative pb-7 pl-6"><span className={cn("absolute -left-1.75 top-1 size-3 rounded-full border-2 border-muted-foreground bg-background", item.current && "border-primary bg-primary shadow-[0_0_12px] shadow-primary/40")} />{item.node}<h3 className="font-medium">{item.title}</h3>{item.meta && <div className="mt-1 text-xs text-muted-foreground">{item.meta}</div>}{item.body && <div className="mt-3">{item.body}</div>}</li>)}</ol>; }

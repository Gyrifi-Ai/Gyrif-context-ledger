import { useReachability } from "../../app/reachability";
import { Button } from "../../ui/primitives/button";
import { cn } from "@/lib/utils";
export function RuntimeStatus() {
	const { health, streamState, streamExhausted, reconnectStream } = useReachability();
	const streamWaiting = health.state !== "offline" && streamState !== "open";
	const label = health.state === "offline" ? "Offline" : streamExhausted ? "Stream closed" : streamWaiting ? "Reconnecting" : health.state === "degraded" ? "Degraded" : "Connected";
	const tone = health.state === "offline" ? "bg-destructive" : streamWaiting || health.state === "degraded" ? "bg-warning" : "bg-success";
	const title = health.version ? `Runtime ${health.version} · inference ${health.inference}` : health.state === "checking" ? "Checking Runtime" : health.state === "offline" ? "Runtime unreachable" : "Runtime connected";
	return <span className="flex items-center justify-between gap-2"><span title={title} className="inline-flex items-center gap-2 text-xs text-muted-foreground"><span className={cn("size-2 rounded-full", tone)} />{label}</span>{streamExhausted && <Button variant="ghost" size="sm" onClick={reconnectStream}>Reconnect</Button>}</span>;
}

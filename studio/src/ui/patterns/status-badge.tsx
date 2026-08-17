import { Badge } from "@/components/ui/badge";
import { statusTone } from "../../features/shared/status";

export function StatusBadge({ value }: { value: string }) {
  return <Badge variant={statusTone(value)}>{value}</Badge>;
}

// Domain status → badge tone, per the normative table in design-system.md §2.2.
export type StatusTone = "neutral" | "info" | "review" | "success" | "warning" | "danger";

const tones: Record<string, StatusTone> = {
  ACCEPTED: "info",
  READY: "neutral",
  INVALID: "danger",
  BLOCKED: "danger",
  RELEASED: "success",
  DRAFT: "neutral",
  REVIEWED: "review",
  APPROVED: "success",
  APPLYING: "warning",
  VERIFYING: "warning",
  FINALIZED: "success",
  RECOVERY_REQUIRED: "danger",
};

export function statusTone(value: string): StatusTone {
  return tones[value.toUpperCase()] ?? "neutral";
}

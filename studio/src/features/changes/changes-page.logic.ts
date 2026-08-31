import type { Change } from "../../api/types";

export type ChangeFilters = {
  status: "ALL" | Change["status"];
  action: "ALL" | Change["action"];
  unit: string;
};

export type ChangeSubmission = {
  unit: string;
  action: Change["action"];
  idempotencyKey: string;
  desired?: unknown;
};

export function filterChangesByUnit(changes: Change[], query: string): Change[] {
  const unit = query.trim().toLocaleLowerCase();
  return unit === "" ? changes : changes.filter((change) => change.unit.toLocaleLowerCase().includes(unit));
}

export function countChangeStatuses(changes: Change[]): { ready: number; released: number; invalid: number } {
  return changes.reduce((counts, change) => {
    if (change.status === "READY") counts.ready += 1;
    if (change.status === "RELEASED") counts.released += 1;
    if (change.status === "INVALID") counts.invalid += 1;
    return counts;
  }, { ready: 0, released: 0, invalid: 0 });
}

export function validateDesiredJson(value: string): string | undefined {
  try {
    JSON.parse(value);
    return undefined;
  } catch (error) {
    return error instanceof Error ? error.message : "Enter valid JSON.";
  }
}

export function buildChangeSubmission(unit: string, action: Change["action"], desired: string, idempotencyKey: string): ChangeSubmission {
  const input: ChangeSubmission = { unit: unit.trim(), action, idempotencyKey: idempotencyKey.trim() };
  if (action === "PUT") input.desired = JSON.parse(desired);
  return input;
}

export function prepareChangeSubmission(unit: string, action: Change["action"], desired: string, idempotencyKey: string): { input?: ChangeSubmission; jsonError?: string } {
  const jsonError = action === "PUT" ? validateDesiredJson(desired) : undefined;
  if (jsonError) return { jsonError };
  return { input: buildChangeSubmission(unit, action, desired, idempotencyKey) };
}

export function moveOrdered(ids: string[], index: number, direction: -1 | 1): string[] {
  const target = index + direction;
  if (target < 0 || target >= ids.length) return ids;
  const reordered = [...ids];
  [reordered[index], reordered[target]] = [reordered[target], reordered[index]];
  return reordered;
}

export function newIdempotencyKey(unit = "change", now = Date.now()): string {
  return `studio-${unit.trim() || "change"}-${now}`;
}
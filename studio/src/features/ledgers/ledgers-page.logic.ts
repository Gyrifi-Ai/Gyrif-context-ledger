import type { Change } from "../../api/types";

export const ledgerNameMaxLength = 120;
export const ledgerDescriptionMaxLength = 500;

type LedgerFormErrors = { name?: string; description?: string };

export function validateLedgerForm(name: string, description: string): LedgerFormErrors {
  const trimmedName = name.trim();
  return {
    name: trimmedName.length === 0
      ? "A ledger name is required."
      : trimmedName.length > ledgerNameMaxLength
        ? `Name must be ${ledgerNameMaxLength} characters or fewer.`
        : undefined,
    description: description.length > ledgerDescriptionMaxLength
      ? `Description must be ${ledgerDescriptionMaxLength} characters or fewer.`
      : undefined,
  };
}

export function countReadyChanges(changes: Change[]): number {
  return changes.filter((change) => change.status === "READY").length;
}
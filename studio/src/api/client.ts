import type { Change, Ledger, Proposal, Release, SystemStatus } from "./types";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { error?: { message?: string } } | null;
    throw new Error(body?.error?.message ?? `Request failed (${response.status})`);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const api = {
  status: () => request<SystemStatus>("/api/v1/system/status"),
  ledgers: () => request<{ items: Ledger[] }>("/api/v1/ledgers"),
  createLedger: (input: { name: string; description?: string }) => request<Ledger>("/api/v1/ledgers", { method: "POST", body: JSON.stringify(input) }),
  changes: (ledgerId: string) => request<{ items: Change[] }>(`/api/v1/ledgers/${ledgerId}/changes`),
  createChange: (ledgerId: string, input: { unit: string; action: "PUT" | "DELETE"; desired?: unknown; idempotencyKey: string }) => request<Change>(`/api/v1/ledgers/${ledgerId}/changes`, { method: "POST", body: JSON.stringify(input) }),
  proposals: (ledgerId: string) => request<{ items: Proposal[] }>(`/api/v1/ledgers/${ledgerId}/proposals`),
  createProposal: (ledgerId: string, input: { title: string; changeIds: string[] }) => request<Proposal>(`/api/v1/ledgers/${ledgerId}/proposals`, { method: "POST", body: JSON.stringify(input) }),
  evaluate: (ledgerId: string, proposalId: string, criteria: string) => request<{ passed: boolean; summary: string }>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/evaluation`, { method: "POST", body: JSON.stringify({ criteria }) }),
  approve: (ledgerId: string, proposalId: string) => request<void>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/approvals`, { method: "POST", body: JSON.stringify({ actor: "local-user" }) }),
  release: (ledgerId: string, proposalId: string) => request<Release>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/release`, { method: "POST" }),
  releases: (ledgerId: string) => request<{ items: Release[] }>(`/api/v1/ledgers/${ledgerId}/releases`),
  rollback: (ledgerId: string, releaseId: string) => request<Proposal>(`/api/v1/ledgers/${ledgerId}/releases/${releaseId}/rollback`, { method: "POST" }),
};

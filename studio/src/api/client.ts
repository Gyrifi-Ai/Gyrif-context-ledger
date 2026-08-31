import type { Approval, Change, CheckResult, EvaluationResponse, Ledger, Proposal, ProposalDetail, Release, ReleaseIntent, ReleaseIntentStatus, RetryReleaseIntentResult, SystemStatus } from "./types";

export type ApiErrorKind = "transport" | "http";

export class ApiError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly status: number,
    readonly kind: ApiErrorKind,
    readonly requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type ErrorEnvelope = { error?: { code?: unknown; message?: unknown } };
type RequestHealth = { reachable: true } | { reachable: false; error: ApiError };
type RequestHealthListener = (health: RequestHealth) => void;

const requestHealthListeners = new Set<RequestHealthListener>();
let lastFailedRequestId: string | undefined;

export function subscribeToRequestHealth(listener: RequestHealthListener): () => void {
  requestHealthListeners.add(listener);
  return () => requestHealthListeners.delete(listener);
}

export function getLastFailedRequestId(): string | undefined {
  return lastFailedRequestId;
}

function reportRequestHealth(health: RequestHealth): void {
  requestHealthListeners.forEach((listener) => listener(health));
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") throw error;
    const transportError = new ApiError("UNAVAILABLE", "Cannot reach the Gyrifi runtime.", 0, "transport");
    reportRequestHealth({ reachable: false, error: transportError });
    throw transportError;
  }

  reportRequestHealth({ reachable: true });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as ErrorEnvelope | null;
    const code = typeof body?.error?.code === "string" ? body.error.code : "UNKNOWN";
    const message = typeof body?.error?.message === "string" ? body.error.message : `Request failed (${response.status})`;
    lastFailedRequestId = response.headers.get("X-Request-ID") ?? undefined;
    throw new ApiError(code, message, response.status, "http", lastFailedRequestId);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const api = {
  status: (init?: RequestInit) => request<SystemStatus>("/api/v1/system/status", init),
  ledgers: (init?: RequestInit) => request<{ items: Ledger[] }>("/api/v1/ledgers", init),
  createLedger: (input: { name: string; description?: string }) => request<Ledger>("/api/v1/ledgers", { method: "POST", body: JSON.stringify(input) }),
  changes: (ledgerId: string, init?: RequestInit) => request<{ items: Change[] }>(`/api/v1/ledgers/${ledgerId}/changes`, init),
  createChange: (ledgerId: string, input: { unit: string; action: "PUT" | "DELETE"; desired?: unknown; idempotencyKey: string }) => request<Change>(`/api/v1/ledgers/${ledgerId}/changes`, { method: "POST", body: JSON.stringify(input) }),
  proposals: (ledgerId: string, init?: RequestInit) => request<{ items: Proposal[] }>(`/api/v1/ledgers/${ledgerId}/proposals`, init),
  createProposal: (ledgerId: string, input: { title: string; changeIds: string[] }) => request<Proposal>(`/api/v1/ledgers/${ledgerId}/proposals`, { method: "POST", body: JSON.stringify(input) }),
  proposal: (ledgerId: string, proposalId: string, init?: RequestInit) => request<ProposalDetail>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}`, init),
  proposalChecks: (ledgerId: string, proposalId: string, init?: RequestInit) => request<{ items: CheckResult[] }>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/checks`, init),
  proposalApprovals: (ledgerId: string, proposalId: string, init?: RequestInit) => request<{ items: Approval[] }>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/approvals`, init),
  evaluate: (ledgerId: string, proposalId: string, criteria: string) => request<EvaluationResponse>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/evaluation`, { method: "POST", body: JSON.stringify({ criteria }) }),
  approve: (ledgerId: string, proposalId: string, actor = "local-user") => request<void>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/approvals`, { method: "POST", body: JSON.stringify({ actor }) }),
  release: (ledgerId: string, proposalId: string) => request<Release>(`/api/v1/ledgers/${ledgerId}/proposals/${proposalId}/release`, { method: "POST" }),
  releaseIntents: (ledgerId: string, status?: ReleaseIntentStatus, init?: RequestInit) => request<{ items: ReleaseIntent[] }>(`/api/v1/ledgers/${ledgerId}/release-intents${status ? `?status=${encodeURIComponent(status)}` : ""}`, init),
  releaseIntent: (ledgerId: string, intentId: string, init?: RequestInit) => request<ReleaseIntent>(`/api/v1/ledgers/${ledgerId}/release-intents/${intentId}`, init),
  retryReleaseIntent: (ledgerId: string, intentId: string) => request<RetryReleaseIntentResult>(`/api/v1/ledgers/${ledgerId}/release-intents/${intentId}/retry`, { method: "POST" }),
  resolveReleaseIntent: (ledgerId: string, intentId: string, note: string) => request<void>(`/api/v1/ledgers/${ledgerId}/release-intents/${intentId}/resolve`, { method: "POST", body: JSON.stringify({ resolution: "ABANDONED", note }) }),
  releases: (ledgerId: string, init?: RequestInit) => request<{ items: Release[] }>(`/api/v1/ledgers/${ledgerId}/releases`, init),
  rollback: (ledgerId: string, releaseId: string) => request<Proposal>(`/api/v1/ledgers/${ledgerId}/releases/${releaseId}/rollback`, { method: "POST" }),
};

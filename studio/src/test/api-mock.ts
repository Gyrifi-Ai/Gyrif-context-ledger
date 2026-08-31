import { vi, type Mock } from "vitest";
import { ApiError, type Api } from "../api/client";
import type { ListOptions } from "../api/types";

export type MockApi = { [K in keyof Api]: Mock<Api[K]> };

export const mockApi: MockApi = {
  status: vi.fn<Api["status"]>(),
  ledgers: vi.fn<Api["ledgers"]>(),
  createLedger: vi.fn<Api["createLedger"]>(),
  changes: vi.fn<Api["changes"]>(),
  createChange: vi.fn<Api["createChange"]>(),
  proposals: vi.fn<Api["proposals"]>(),
  createProposal: vi.fn<Api["createProposal"]>(),
  proposal: vi.fn<Api["proposal"]>(),
  proposalChecks: vi.fn<Api["proposalChecks"]>(),
  proposalApprovals: vi.fn<Api["proposalApprovals"]>(),
  evaluate: vi.fn<Api["evaluate"]>(),
  approve: vi.fn<Api["approve"]>(),
  cancelProposal: vi.fn<Api["cancelProposal"]>(),
  release: vi.fn<Api["release"]>(),
  releaseIntents: vi.fn<Api["releaseIntents"]>(),
  releaseIntent: vi.fn<Api["releaseIntent"]>(),
  retryReleaseIntent: vi.fn<Api["retryReleaseIntent"]>(),
  resolveReleaseIntent: vi.fn<Api["resolveReleaseIntent"]>(),
  releases: vi.fn<Api["releases"]>(),
  rollback: vi.fn<Api["rollback"]>(),
};

function body(init?: RequestInit): Record<string, unknown> {
  return init?.body ? JSON.parse(String(init.body)) as Record<string, unknown> : {};
}

function response(value: unknown): Response {
  return value === undefined
    ? new Response(null, { status: 204 })
    : new Response(JSON.stringify(value), { status: 200, headers: { "Content-Type": "application/json" } });
}

function listOptions(url: URL): ListOptions | undefined {
  const options: ListOptions = {};
  const limit = url.searchParams.get("limit");
  if (limit !== null) options.limit = Number(limit);
  const cursor = url.searchParams.get("cursor");
  if (cursor !== null) options.cursor = cursor;
  const status = url.searchParams.get("status");
  if (status !== null) options.status = status;
  const action = url.searchParams.get("action");
  if (action !== null) options.action = action;
  return Object.keys(options).length > 0 ? options : undefined;
}

async function route(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, "http://studio.test");
  const path = url.pathname;
  const method = init?.method ?? "GET";
  const match = (pattern: RegExp) => pattern.exec(path)?.slice(1).map(decodeURIComponent);
  try {
    if (method === "GET" && path === "/api/v1/system/status") return response(await mockApi.status(init));
    if (method === "GET" && path === "/api/v1/ledgers") return response(await mockApi.ledgers(listOptions(url), init));
    if (method === "POST" && path === "/api/v1/ledgers") return response(await mockApi.createLedger(body(init) as Parameters<Api["createLedger"]>[0]));
    let values = match(/^\/api\/v1\/ledgers\/([^/]+)\/changes$/);
    if (values && method === "GET") return response(await mockApi.changes(values[0], listOptions(url), init));
    if (values && method === "POST") return response(await mockApi.createChange(values[0], body(init) as Parameters<Api["createChange"]>[1]));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals$/);
    if (values && method === "GET") return response(await mockApi.proposals(values[0], listOptions(url), init));
    if (values && method === "POST") return response(await mockApi.createProposal(values[0], body(init) as Parameters<Api["createProposal"]>[1]));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)$/);
    if (values && method === "GET") return response(await mockApi.proposal(values[0], values[1], init));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)\/checks$/);
    if (values && method === "GET") return response(await mockApi.proposalChecks(values[0], values[1], init));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)\/approvals$/);
    if (values && method === "GET") return response(await mockApi.proposalApprovals(values[0], values[1], init));
    if (values && method === "POST") return response(await mockApi.approve(values[0], values[1], String(body(init).actor ?? "")));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)\/cancel$/);
    if (values && method === "POST") return response(await mockApi.cancelProposal(values[0], values[1]));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)\/evaluation$/);
    if (values && method === "POST") return response(await mockApi.evaluate(values[0], values[1], String(body(init).criteria ?? "")));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/proposals\/([^/]+)\/release$/);
    if (values && method === "POST") return response(await mockApi.release(values[0], values[1]));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/release-intents$/);
    if (values && method === "GET") return response(await mockApi.releaseIntents(values[0], (url.searchParams.get("status") ?? undefined) as Parameters<Api["releaseIntents"]>[1], init));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/release-intents\/([^/]+)$/);
    if (values && method === "GET") return response(await mockApi.releaseIntent(values[0], values[1], init));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/release-intents\/([^/]+)\/retry$/);
    if (values && method === "POST") return response(await mockApi.retryReleaseIntent(values[0], values[1]));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/release-intents\/([^/]+)\/resolve$/);
    if (values && method === "POST") return response(await mockApi.resolveReleaseIntent(values[0], values[1], String(body(init).note ?? "")));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/releases$/);
    if (values && method === "GET") return response(await mockApi.releases(values[0], listOptions(url), init));
    values = match(/^\/api\/v1\/ledgers\/([^/]+)\/releases\/([^/]+)\/rollback$/);
    if (values && method === "POST") return response(await mockApi.rollback(values[0], values[1]));
    throw new Error(`Unexpected API request: ${method} ${path}`);
  } catch (error) {
    if (error instanceof ApiError && error.kind === "http") {
      return new Response(JSON.stringify({ error: { code: error.code, message: error.message } }), { status: error.status, headers: { "Content-Type": "application/json", "X-Request-ID": error.requestId ?? "" } });
    }
    throw error;
  }
}

export function resetApiMock(): void {
  Object.values(mockApi).forEach((mock) => mock.mockReset());
  mockApi.status.mockResolvedValue({ status: "ok", version: "test", commit: "test-commit", buildDate: "2026-09-01T00:00:00Z", inference: "disabled", health: { database: "ok", target: "unknown", inference: "disabled", unresolvedIntents: 0 } });
  mockApi.ledgers.mockResolvedValue({ items: [] });
  mockApi.changes.mockResolvedValue({ items: [] });
  mockApi.proposals.mockResolvedValue({ items: [] });
  mockApi.proposalChecks.mockResolvedValue({ items: [] });
  mockApi.proposalApprovals.mockResolvedValue({ items: [] });
  mockApi.releaseIntents.mockResolvedValue({ items: [] });
  mockApi.releases.mockResolvedValue({ items: [] });
  vi.stubGlobal("fetch", vi.fn(route));
}
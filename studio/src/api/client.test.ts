import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api, subscribeToRequestHealth } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("API client", () => {
  it("uses the versioned ledger endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.ledgers()).resolves.toEqual({ items: [] });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/ledgers", expect.any(Object));
  });

  it("uses the proposal detail and evidence endpoints", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json" } })));
    vi.stubGlobal("fetch", fetchMock);

    await api.proposal("ldg one", "pr/one");
    await api.proposalChecks("ldg one", "pr/one");
    await api.proposalApprovals("ldg one", "pr/one");

    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      "/api/v1/ledgers/ldg one/proposals/pr/one",
      "/api/v1/ledgers/ldg one/proposals/pr/one/checks",
      "/api/v1/ledgers/ldg one/proposals/pr/one/approvals",
    ]);
  });

  it("sends the user-selected approving actor", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await api.approve("ldg_one", "pr_one", "reviewer@example.com");
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/ledgers/ldg_one/proposals/pr_one/approvals", expect.objectContaining({ method: "POST", body: JSON.stringify({ actor: "reviewer@example.com" }) }));
  });

  it("maps structured API errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "CONFLICT", message: "blocked" } }), { status: 409, headers: { "X-Request-ID": "req-123" } })));
    const error = await api.createLedger({ name: "duplicate" }).catch((value: unknown) => value);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ code: "CONFLICT", message: "blocked", status: 409, kind: "http", requestId: "req-123" } satisfies Partial<ApiError>);
  });

  it("uses UNKNOWN for malformed error envelopes", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("not json", { status: 500 })));
    const error = await api.ledgers().catch((value: unknown) => value);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ code: "UNKNOWN", message: "Request failed (500)", status: 500, kind: "http" } satisfies Partial<ApiError>);
  });

  it("classifies rejected fetches as transport failures and reports recovery", async () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToRequestHealth(listener);
    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new TypeError("connection refused")));

    const error = await api.ledgers().catch((value: unknown) => value);

    expect(error).toMatchObject({ code: "UNAVAILABLE", status: 0, kind: "transport" } satisfies Partial<ApiError>);
    expect(listener).toHaveBeenLastCalledWith({ reachable: false, error });

    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200 })));
    await api.ledgers();
    expect(listener).toHaveBeenLastCalledWith({ reachable: true });
    unsubscribe();
  });

  it("keeps a responding 503 classified as reachable HTTP failure", async () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToRequestHealth(listener);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "UNAVAILABLE", message: "Target unavailable" } }), { status: 503 })));

    const error = await api.ledgers().catch((value: unknown) => value);

    expect(error).toMatchObject({ status: 503, kind: "http" } satisfies Partial<ApiError>);
    expect(listener).toHaveBeenLastCalledWith({ reachable: true });
    unsubscribe();
  });
});

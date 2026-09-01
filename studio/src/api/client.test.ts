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

  it("encodes list options and propagates next cursors", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [], nextCursor: "next-page" }), { status: 200, headers: { "Content-Type": "application/json" } })));
    vi.stubGlobal("fetch", fetchMock);

    await expect(api.changes("ldg_one", { limit: 25, cursor: "a/b+c=", status: "READY", action: "DELETE" })).resolves.toEqual({ items: [], nextCursor: "next-page" });
    await api.proposals("ldg_one", { cursor: "proposal cursor", status: "CANCELLED" });
    await api.releases("ldg_one", { limit: 1 });
    await api.ledgers({ cursor: "ledger cursor" });

    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      "/api/v1/ledgers/ldg_one/changes?limit=25&cursor=a%2Fb%2Bc%3D&status=READY&action=DELETE",
      "/api/v1/ledgers/ldg_one/proposals?cursor=proposal+cursor&status=CANCELLED",
      "/api/v1/ledgers/ldg_one/releases?limit=1",
      "/api/v1/ledgers?cursor=ledger+cursor",
    ]);
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

  it("posts Proposal cancellation without a request body", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await api.cancelProposal("ldg_one", "pr_one");
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/ledgers/ldg_one/proposals/pr_one/cancel", expect.objectContaining({ method: "POST" }));
    expect(fetchMock.mock.calls[0][1]).not.toHaveProperty("body");
  });

  it("uses the lifecycle endpoints and archived list flag", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await api.withdrawChange("ldg_one", "chg_one", "wrong source");
    await api.archiveLedger("ldg_one");
    await api.unarchiveLedger("ldg_one");
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    await api.ledgers({ includeArchived: true });
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      "/api/v1/ledgers/ldg_one/changes/chg_one/withdraw",
      "/api/v1/ledgers/ldg_one/archive",
      "/api/v1/ledgers/ldg_one/unarchive",
      "/api/v1/ledgers?includeArchived=true",
    ]);
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: "POST", body: JSON.stringify({ reason: "wrong source" }) });
  });

  it("uses the Release Intent inspection and recovery endpoints", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json" } })));
    vi.stubGlobal("fetch", fetchMock);
    await api.releaseIntents("ldg_one", "RECOVERY_REQUIRED");
    await api.releaseIntent("ldg_one", "intent_one");
    await api.retryReleaseIntent("ldg_one", "intent_one");
    await api.resolveReleaseIntent("ldg_one", "intent_one", "inspected target");
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      "/api/v1/ledgers/ldg_one/release-intents?status=RECOVERY_REQUIRED",
      "/api/v1/ledgers/ldg_one/release-intents/intent_one",
      "/api/v1/ledgers/ldg_one/release-intents/intent_one/retry",
      "/api/v1/ledgers/ldg_one/release-intents/intent_one/resolve",
    ]);
    expect(fetchMock.mock.calls[3][1]).toMatchObject({ method: "POST", body: JSON.stringify({ resolution: "ABANDONED", note: "inspected target" }) });
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

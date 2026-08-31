import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("API client", () => {
  it("uses the versioned ledger endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.ledgers()).resolves.toEqual({ items: [] });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/ledgers", expect.any(Object));
  });

  it("maps structured API errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "CONFLICT", message: "blocked" } }), { status: 409 })));
    const error = await api.createLedger({ name: "duplicate" }).catch((value: unknown) => value);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ code: "CONFLICT", message: "blocked", status: 409 } satisfies Partial<ApiError>);
  });

  it("uses UNKNOWN for malformed error envelopes", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("not json", { status: 500 })));
    const error = await api.ledgers().catch((value: unknown) => value);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ code: "UNKNOWN", message: "Request failed (500)", status: 500 } satisfies Partial<ApiError>);
  });
});

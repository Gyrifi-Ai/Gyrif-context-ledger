import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("API client", () => {
  it("uses the versioned ledger endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.ledgers()).resolves.toEqual({ items: [] });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/ledgers", expect.any(Object));
  });

  it("maps structured API errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { message: "blocked" } }), { status: 409 })));
    await expect(api.createLedger({ name: "duplicate" })).rejects.toThrow("blocked");
  });
});

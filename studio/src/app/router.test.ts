import { describe, expect, it } from "vitest";
import { parseRoute } from "./router";

describe("parseRoute", () => {
  it("parses top-level and linkable Proposal routes", () => {
    expect(parseRoute("#changes")).toEqual({ area: "changes" });
    expect(parseRoute("#proposals/pr_123")).toEqual({ area: "proposals", id: "pr_123" });
  });

  it("falls back for malformed routes", () => {
    expect(parseRoute("#proposals/pr_123/extra")).toEqual({ area: "ledgers" });
    expect(parseRoute("#unknown")).toEqual({ area: "ledgers" });
  });
});

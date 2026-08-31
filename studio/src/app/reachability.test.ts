import { describe, expect, it } from "vitest";
import { reachabilityRetryDelay } from "./reachability";

describe("reachability backoff", () => {
  it("starts at one second and caps at thirty seconds", () => {
    expect([1, 2, 3, 4, 5, 6, 7].map(reachabilityRetryDelay)).toEqual([1_000, 2_000, 4_000, 8_000, 16_000, 30_000, 30_000]);
  });
});
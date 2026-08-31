import { describe, expect, it } from "vitest";
import { formatAge } from "./time";

const now = Date.parse("2026-08-31T12:00:00Z");

describe("formatAge", () => {
  it("formats minute, hour, and day ages", () => {
    expect(formatAge("2026-08-31T11:58:00Z", now)).toBe("2m ago");
    expect(formatAge("2026-08-31T09:00:00Z", now)).toBe("3h ago");
    expect(formatAge("2026-08-28T12:00:00Z", now)).toBe("3d ago");
  });

  it("handles current, future, and malformed values", () => {
    expect(formatAge("2026-08-31T12:00:00Z", now)).toBe("now");
    expect(formatAge("2026-08-31T12:01:00Z", now)).toBe("now");
    expect(formatAge("invalid", now)).toBe("Unknown");
  });
});
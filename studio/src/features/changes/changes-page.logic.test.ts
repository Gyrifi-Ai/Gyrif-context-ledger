import { describe, expect, it } from "vitest";
import type { Change } from "../../api/types";
import { buildChangeSubmission, countChangeStatuses, filterChanges, moveOrdered, newIdempotencyKey, prepareChangeSubmission, validateDesiredJson } from "./changes-page.logic";

const changes = [
  { id: "one", unit: "point/Alpha", action: "PUT", status: "READY" },
  { id: "two", unit: "point/beta", action: "DELETE", status: "RELEASED" },
  { id: "three", unit: "archive/alpha", action: "DELETE", status: "INVALID" },
] as Change[];

describe("Changes page logic", () => {
  it("filters by status, action, and case-insensitive unit text", () => {
    expect(filterChanges(changes, { status: "ALL", action: "DELETE", unit: "ALPHA" }).map((change) => change.id)).toEqual(["three"]);
    expect(filterChanges(changes, { status: "READY", action: "ALL", unit: "point" }).map((change) => change.id)).toEqual(["one"]);
  });

  it("counts the three inbox status metrics", () => {
    expect(countChangeStatuses(changes)).toEqual({ ready: 1, released: 1, invalid: 1 });
  });

  it("returns the JSON parser message for invalid desired state", () => {
    expect(validateDesiredJson("{")) .toMatch(/JSON|property|position|end/i);
    expect(validateDesiredJson(`{"ok":true}`)).toBeUndefined();
  });

  it("omits desired from DELETE submissions", () => {
    expect(prepareChangeSubmission(" point/42 ", "DELETE", "not-json", " delete-42 ")).toEqual({ input: { unit: "point/42", action: "DELETE", idempotencyKey: "delete-42" } });
    expect(buildChangeSubmission("point/42", "PUT", `{"ok":true}`, "put-42")).toEqual({ unit: "point/42", action: "PUT", idempotencyKey: "put-42", desired: { ok: true } });
  });

  it("blocks invalid PUT JSON before building a request", () => {
    const prepared = prepareChangeSubmission("point/42", "PUT", "{", "put-42");
    expect(prepared.input).toBeUndefined();
    expect(prepared.jsonError).toMatch(/JSON|property|position|end/i);
  });

  it("reorders selected IDs without mutating the input", () => {
    const original = ["a", "b", "c"];
    expect(moveOrdered(original, 1, -1)).toEqual(["b", "a", "c"]);
    expect(moveOrdered(original, 1, 1)).toEqual(["a", "c", "b"]);
    expect(moveOrdered(original, 0, -1)).toBe(original);
    expect(original).toEqual(["a", "b", "c"]);
  });

  it("generates a visible stable-format default idempotency key", () => {
    expect(newIdempotencyKey("point-42", 1234)).toBe("studio-point-42-1234");
    expect(newIdempotencyKey("", 1234)).toBe("studio-change-1234");
  });
});
import { describe, expect, it } from "vitest";
import { changeTone, intentTone, proposalTone } from "./status";

describe("domain status tones", () => {
  it("maps every API status to its normative tone", () => {
    expect(changeTone("ACCEPTED")).toBe("info"); expect(changeTone("READY")).toBe("neutral"); expect(changeTone("INVALID")).toBe("danger"); expect(changeTone("RELEASED")).toBe("success");
    expect(proposalTone("DRAFT")).toBe("neutral"); expect(proposalTone("REVIEWED")).toBe("review"); expect(proposalTone("APPROVED")).toBe("success"); expect(proposalTone("RELEASED")).toBe("success"); expect(proposalTone("BLOCKED")).toBe("danger");
    expect(intentTone("READY")).toBe("warning"); expect(intentTone("APPLYING")).toBe("warning"); expect(intentTone("VERIFYING")).toBe("warning"); expect(intentTone("FINALIZED")).toBe("success"); expect(intentTone("RECOVERY_REQUIRED")).toBe("danger");
  });
});

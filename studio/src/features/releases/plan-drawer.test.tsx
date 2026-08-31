import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { ReleaseIntentOperation } from "../../api/types";
import { PlanDrawer } from "./plan-drawer";

describe("PlanDrawer", () => {
  it("shows fingerprints, target metric, and flags only missing required before-images", () => {
    const operations: ReleaseIntentOperation[] = [
      { unit: "point/missing", action: "PUT", expectedFingerprint: "sha256:before", expectedExists: true, desiredFingerprint: "sha256:desired", targetMetric: "Cosine", beforeExists: true, beforeObjectHash: "sha256:object", hasBeforeImage: false },
      { unit: "point/new", action: "PUT", expectedExists: false, desiredFingerprint: "sha256:new", beforeExists: false, hasBeforeImage: false },
    ];
    const html = renderToStaticMarkup(<PlanDrawer open onClose={() => undefined} release={{ id: "rel_one_123456", ledgerId: "ldg_one", proposalId: "pr_one", hash: "sha256:release", createdAt: "2026-08-31T00:00:00Z" }} operations={operations} />);
    expect(html).toContain("point/missing");
    expect(html).toContain("sha256:before");
    expect(html).toContain("sha256:desired");
    expect(html).toContain("Target metric: Cosine");
    expect(html.match(/No rollback material for this unit\./g)).toHaveLength(1);
    expect(html).toContain("No prior value; rollback will delete this unit.");
  });
});
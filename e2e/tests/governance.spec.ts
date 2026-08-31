import type { Page } from "@playwright/test";
import { expect, ingestChange, qdrantPoint, qdrantPoints, runtimeURL, startRuntime, terminateRuntime, test } from "./harness";

type Ledger = { id: string; name: string };
type Proposal = { id: string; title: string };
type Release = { id: string; proposalId: string };

async function createLedger(page: Page, name: string): Promise<Ledger> {
  await page.goto("/#ledgers");
  const create = page.getByRole("button", { name: "Create your first ledger" });
  await expect(create).toBeVisible();
  await create.click();
  const dialog = page.getByRole("dialog", { name: "Create ledger" });
  await dialog.getByLabel("Name").fill(name);
  await dialog.getByLabel("Description").fill("Browser qualification ledger");
  const responsePromise = page.waitForResponse((response) => response.request().method() === "POST" && new URL(response.url()).pathname === "/api/v1/ledgers");
  await dialog.getByRole("button", { name: "Create ledger" }).click();
  const response = await responsePromise;
  expect(response.status()).toBe(201);
  const ledger = await response.json() as Ledger;
  await expect(page.getByRole("status")).toContainText(name);
  return ledger;
}

async function createProposal(page: Page, changeIDs: string[], title: string): Promise<Proposal> {
  await page.goto("/#changes");
  await expect(page.getByRole("heading", { name: "Changes" })).toBeVisible();
  for (const changeID of changeIDs) {
    await page.getByRole("checkbox", { name: `Select ${changeID}` }).check();
  }
  await page.getByRole("button", { name: "Create proposal" }).click();
  const dialog = page.getByRole("dialog", { name: "Create proposal" });
  await dialog.getByLabel("Title").fill(title);
  const responsePromise = page.waitForResponse((response) => response.request().method() === "POST" && /\/api\/v1\/ledgers\/[^/]+\/proposals$/.test(new URL(response.url()).pathname));
  await dialog.getByRole("button", { name: "Create proposal" }).click();
  const response = await responsePromise;
  expect(response.status()).toBe(201);
  const proposal = await response.json() as Proposal;
  await expect(page).toHaveURL(new RegExp(`#proposals/${proposal.id}$`));
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  return proposal;
}

async function completeProposal(page: Page, title: string, assertInitialGates: boolean): Promise<Release> {
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  const approve = page.getByRole("button", { name: "Approve" });
  const release = page.getByRole("button", { name: "Release to Qdrant" });

  if (assertInitialGates) {
    await expect(approve).toBeDisabled();
    await expect(approve).toHaveAttribute("title", "A current passing evaluation is required before approval.");
    await expect(page.getByText("A current passing evaluation is required before approval.")).toBeVisible();
    await expect(release).toBeDisabled();
    await expect(release).toHaveAttribute("title", "A current passing evaluation is required.");
    await expect(page.getByText("A current passing evaluation is required.", { exact: true })).toBeVisible();
  }

  await page.getByLabel("Evaluation criteria").fill("The effective context must be internally consistent and safe to release.");
  const evaluationPromise = page.waitForResponse((response) => response.request().method() === "POST" && new URL(response.url()).pathname.endsWith("/evaluation"));
  await page.getByRole("button", { name: "Run evaluation" }).click();
  expect((await evaluationPromise).status()).toBe(200);
  const evidence = page.getByRole("article", { name: "Evaluation evidence" });
  await expect(evidence).toContainText("PASS");
  await expect(evidence).toContainText("DETERMINISTIC");
  await expect(evidence).toContainText("Deterministic proposal checks passed; local natural-language evaluation is disabled.");
  await expect(evidence).toContainText("This check was deterministic. Natural-language evaluation is off.");

  await expect(release).toBeDisabled();
  await expect(release).toHaveAttribute("title", "A current approval is required.");
  await page.getByLabel("Approving actor").fill("e2e-operator");
  const approvalPromise = page.waitForResponse((response) => response.request().method() === "POST" && new URL(response.url()).pathname.endsWith("/approvals"));
  await approve.click();
  expect((await approvalPromise).status()).toBe(204);
  await expect(page.getByText("e2e-operator")).toBeVisible();
  await expect(release).toBeEnabled();

  await release.click();
  const dialog = page.getByRole("dialog", { name: "Release to Qdrant?" });
  const responsePromise = page.waitForResponse((response) => response.request().method() === "POST" && new URL(response.url()).pathname.endsWith("/release"));
  await dialog.getByRole("button", { name: "Release to Qdrant" }).click();
  const response = await responsePromise;
  expect(response.status()).toBe(201);
  const result = await response.json() as Release;
  await expect(page.getByText("Release recorded after verification; HEAD has advanced.")).toBeVisible();
  return result;
}

function expectCosineDirection(actual: number[], expected: number[]): void {
  const dot = actual.reduce((sum, value, index) => sum + value * expected[index], 0);
  const left = Math.sqrt(actual.reduce((sum, value) => sum + value * value, 0));
  const right = Math.sqrt(expected.reduce((sum, value) => sum + value * value, 0));
  expect(dot / (left * right)).toBeCloseTo(1, 5);
}

function releaseChip(page: Page, releaseID: string) {
  return page.getByRole("button", { name: `Release · ${releaseID.slice(0, 10)}…` });
}

async function qualifyCanonicalViewports(page: Page, proposal: Proposal, visibleUnit: string): Promise<void> {
  for (const width of [1440, 1180, 900, 480]) {
    await page.setViewportSize({ width, height: 1024 });
    await page.goto("/#changes");
    await expect(page.getByRole("heading", { name: "Changes" })).toBeVisible();
    await expect(page.getByRole("cell", { name: visibleUnit, exact: true })).toBeVisible();
    await page.goto(`/#proposals/${proposal.id}`);
    await expect(page.getByRole("heading", { name: proposal.title })).toBeVisible();
    await expect(page.getByRole("button", { name: "Approve" })).toBeDisabled();
    await expect(page.getByText("A current passing evaluation is required before approval.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Release to Qdrant" })).toBeDisabled();
    await expect(page.getByText("A current passing evaluation is required.", { exact: true })).toBeVisible();
    const documentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    expect(documentWidth).toBeLessThanOrEqual(width);
  }
}

test("a fresh shipped image shows the empty first-run path", async ({ page, stack }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Ledgers" })).toBeVisible();
  await expect(page.getByText("No Ledgers yet")).toBeVisible();
  await expect(page.getByRole("button", { name: "Create your first ledger" })).toBeVisible();
  await expect(qdrantPoints(stack.collection)).resolves.toEqual([]);

  await createLedger(page, `Empty path ${Date.now()}`);
});

test("the shipped image governs, rolls back, restarts, and deep-links", async ({ page, context, request, stack }) => {
  const ledgerName = `Governance ${Date.now()}`;
  const ledger = await createLedger(page, ledgerName);
  const firstDesired = [
    { id: 1001, vector: [1, 0, 0], payload: { version: "first", text: "Original alpha context" } },
    { id: 1002, vector: [0, 1, 0], payload: { version: "first", text: "Original beta context" } },
  ];
  const firstChanges = await Promise.all(firstDesired.map((desired, index) => ingestChange(request, ledger.id, {
    unit: String(desired.id),
    desired,
    idempotencyKey: `e2e-first-${index}-${Date.now()}`,
  })));

  const firstTitle = "Initial context release";
  const firstProposal = await createProposal(page, firstChanges.map((change) => change.id), firstTitle);
  await qualifyCanonicalViewports(page, firstProposal, String(firstDesired[0].id));
  const firstRelease = await completeProposal(page, firstTitle, true);

  await page.goto("/#releases");
  await expect(page.getByText(firstTitle)).toBeVisible();
  await expect(page.getByText("HEAD", { exact: true })).toBeVisible();
  await expect(releaseChip(page, firstRelease.id)).toBeVisible();
  for (const desired of firstDesired) {
    const point = await qdrantPoint(stack.collection, desired.id);
    expect(point?.payload).toEqual(desired.payload);
    expectCosineDirection(point?.vector ?? [], desired.vector);
  }

  const secondDesired = [
    { id: 1001, vector: [0, 0, 1], payload: { version: "second", text: "Updated alpha context" } },
    { id: 1002, vector: [1, 1, 0], payload: { version: "second", text: "Updated beta context" } },
  ];
  const secondChanges = await Promise.all(secondDesired.map((desired, index) => ingestChange(request, ledger.id, {
    unit: String(desired.id),
    desired,
    idempotencyKey: `e2e-second-${index}-${Date.now()}`,
  })));
  const secondTitle = "Updated context release";
  await createProposal(page, secondChanges.map((change) => change.id), secondTitle);
  const secondRelease = await completeProposal(page, secondTitle, false);

  await page.goto("/#releases");
  await expect(page.getByText(secondTitle)).toBeVisible();
  await expect(releaseChip(page, secondRelease.id)).toBeVisible();
  for (const desired of secondDesired) {
    const point = await qdrantPoint(stack.collection, desired.id);
    expect(point?.payload).toEqual(desired.payload);
    expectCosineDirection(point?.vector ?? [], desired.vector);
  }

  await page.getByRole("button", { name: "Roll back to here" }).click();
  const rollbackDialog = page.getByRole("dialog", { name: "Create rollback proposal?" });
  await expect(rollbackDialog).toContainText("This creates a new proposal; it does not rewind history.");
  await expect(rollbackDialog).toContainText("2 units will be restored to their state at this release.");
  await expect(rollbackDialog).toContainText("The proposal must be evaluated, approved, and released like any other.");
  await expect(rollbackDialog).toContainText("HEAD will move forward to a new release after verification.");
  const rollbackPromise = page.waitForResponse((response) => response.request().method() === "POST" && new URL(response.url()).pathname.endsWith(`/releases/${firstRelease.id}/rollback`));
  await rollbackDialog.getByRole("button", { name: "Create rollback proposal" }).click();
  const rollbackResponse = await rollbackPromise;
  expect(rollbackResponse.status()).toBe(201);
  const rollbackProposal = await rollbackResponse.json() as Proposal;
  await page.getByRole("link", { name: "Review proposal" }).click();
  await expect(page).toHaveURL(new RegExp(`#proposals/${rollbackProposal.id}$`));
  const thirdRelease = await completeProposal(page, rollbackProposal.title, false);
  expect(thirdRelease.id).not.toBe(firstRelease.id);
  expect(thirdRelease.id).not.toBe(secondRelease.id);

  for (const desired of firstDesired) {
    const point = await qdrantPoint(stack.collection, desired.id);
    expect(point?.payload).toEqual(desired.payload);
    expectCosineDirection(point?.vector ?? [], desired.vector);
  }

  await page.goto("/#releases");
  await expect(page.getByText("3 immutable releases, newest first.")).toBeVisible();
  await expect(releaseChip(page, thirdRelease.id)).toBeVisible();
  await expect(page.getByText("HEAD", { exact: true })).toBeVisible();

  const shutdown = await terminateRuntime(stack.collection);
  expect(shutdown.exitCode).toBe(0);
  expect(shutdown.afterWal).toBeLessThanOrEqual(shutdown.beforeWal);
  expect(shutdown.afterWal).toBeLessThan(4 * 1024 * 1024);
  await startRuntime(stack.collection);

  await page.reload();
  await expect(page.getByRole("heading", { name: "Releases" })).toBeVisible();
  await expect(page.getByText("3 immutable releases, newest first.")).toBeVisible();
  await expect(releaseChip(page, thirdRelease.id)).toBeVisible();
  await page.goto("/#ledgers");
  await expect(page.getByRole("button", { name: ledgerName, exact: true })).toBeVisible();
  await expect(page.getByText("3 releases")).toBeVisible();
  await page.goto("/#changes");
  await expect(page.getByRole("row")).toHaveCount(7);
  await page.goto("/#proposals");
  await expect(page.getByText(firstProposal.title)).toBeVisible();
  await expect(page.getByText(secondTitle)).toBeVisible();
  await expect(page.getByRole("heading", { name: rollbackProposal.title })).toBeVisible();

  const changesDeepLink = await context.newPage();
  await changesDeepLink.goto(`${runtimeURL}/operator#changes`);
  await expect(changesDeepLink.getByRole("heading", { name: "Changes" })).toBeVisible();
  await expect(changesDeepLink.getByRole("row")).toHaveCount(7);
  const proposalDeepLink = await context.newPage();
  await proposalDeepLink.goto(`${runtimeURL}/operator#proposals/${firstProposal.id}`);
  await expect(proposalDeepLink.getByRole("heading", { name: firstTitle })).toBeVisible();
  await expect(proposalDeepLink.getByRole("article", { name: "Evaluation evidence" })).toContainText("DETERMINISTIC");
});

import { execFile } from "node:child_process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import { expect, test as base, type APIRequestContext } from "@playwright/test";

const execFileAsync = promisify(execFile);
const composeFile = fileURLToPath(new URL("../compose.yaml", import.meta.url));
const projectName = "gyrifi-e2e";
const runtimePort = process.env.GYRIFI_E2E_PORT ?? "18082";
const qdrantPort = process.env.GYRIFI_E2E_QDRANT_PORT ?? "16333";

export const runtimeURL = `http://127.0.0.1:${runtimePort}`;
export const qdrantURL = `http://127.0.0.1:${qdrantPort}`;

function environment(collection: string): NodeJS.ProcessEnv {
  return {
    ...process.env,
    COMPOSE_PROJECT_NAME: projectName,
    GYRIFI_E2E_COLLECTION: collection,
    GYRIFI_E2E_PORT: runtimePort,
    GYRIFI_E2E_QDRANT_PORT: qdrantPort,
  };
}

async function command(file: string, args: string[], collection: string): Promise<string> {
  try {
    const result = await execFileAsync(file, args, {
      env: environment(collection),
      timeout: 240_000,
      maxBuffer: 8 * 1024 * 1024,
    });
    return result.stdout.trim();
  } catch (error) {
    const detail = error as Error & { stdout?: string; stderr?: string };
    throw new Error(`${file} ${args.join(" ")} failed: ${detail.message}\n${detail.stdout ?? ""}\n${detail.stderr ?? ""}`);
  }
}

async function compose(args: string[], collection: string): Promise<string> {
  return command("docker", ["compose", "--file", composeFile, ...args], collection);
}

export async function buildImage(): Promise<void> {
  await compose(["build", "gyrifi"], "gyrifi_bootstrap");
}

export async function stopStack(collection: string): Promise<void> {
  await compose(["down", "--volumes", "--remove-orphans"], collection).catch(() => undefined);
}

export async function waitForHealth(timeoutMs = 45_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError = "no response";
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${runtimeURL}/api/v1/system/status`);
      if (response.ok) {
        const status = await response.json() as { status?: string };
        if (status.status === "ok") return;
        lastError = `status=${String(status.status)}`;
      } else {
        lastError = `HTTP ${response.status}`;
      }
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error);
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`Runtime did not become healthy within ${timeoutMs}ms: ${lastError}`);
}

export async function startFreshStack(collection: string): Promise<void> {
  await stopStack(collection);
  await compose(["up", "--detach", "--no-build", "--wait"], collection);
  await waitForHealth();
}

export async function startRuntime(collection: string): Promise<void> {
  await compose(["start", "gyrifi"], collection);
  await waitForHealth();
}

async function runtimeContainer(collection: string): Promise<string> {
  const id = await compose(["ps", "--quiet", "gyrifi"], collection);
  if (!id) throw new Error("Gyrifi container is not available");
  return id;
}

export async function walSize(collection: string, running: boolean): Promise<number> {
  const shell = "if [ -e /data/state.db-wal ]; then stat -c %s /data/state.db-wal; else echo 0; fi";
  const output = running
    ? await compose(["exec", "--no-TTY", "gyrifi", "sh", "-c", shell], collection)
    : await compose(["run", "--rm", "--no-deps", "--entrypoint", "sh", "gyrifi", "-c", shell], collection);
  const match = output.match(/(\d+)\s*$/);
  if (!match) throw new Error(`Could not read WAL size from: ${output}`);
  return Number(match[1]);
}

export async function terminateRuntime(collection: string): Promise<{ beforeWal: number; afterWal: number; exitCode: number }> {
  const beforeWal = await walSize(collection, true);
  const container = await runtimeContainer(collection);
  await command("docker", ["kill", "--signal", "TERM", container], collection);
  const waited = await command("docker", ["wait", container], collection);
  const exitCode = Number(waited.trim());
  const afterWal = await walSize(collection, false);
  return { beforeWal, afterWal, exitCode };
}

export async function qdrantPoint(collection: string, pointID: number): Promise<{ id: number; vector: number[]; payload: Record<string, unknown> } | null> {
  const response = await fetch(`${qdrantURL}/collections/${encodeURIComponent(collection)}/points/${pointID}`);
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`Qdrant point read failed with HTTP ${response.status}`);
  const body = await response.json() as { result: { id: number; vector: number[]; payload: Record<string, unknown> } };
  return body.result;
}

export async function qdrantPoints(collection: string): Promise<unknown[]> {
  const response = await fetch(`${qdrantURL}/collections/${encodeURIComponent(collection)}/points/scroll`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ limit: 100, with_payload: true, with_vector: true }),
  });
  if (!response.ok) throw new Error(`Qdrant scroll failed with HTTP ${response.status}`);
  const body = await response.json() as { result: { points: unknown[] } };
  return body.result.points;
}

export async function ingestChange(request: APIRequestContext, ledgerID: string, input: { unit: string; desired: Record<string, unknown>; idempotencyKey: string }): Promise<{ id: string }> {
  const response = await request.post(`${runtimeURL}/api/v1/ledgers/${ledgerID}/changes`, {
    data: { ...input, action: "PUT" },
  });
  expect(response.status()).toBe(202);
  return response.json() as Promise<{ id: string }>;
}

export type Stack = { collection: string };

export const test = base.extend<{ stack: Stack }>({
  stack: async ({}, use, testInfo) => {
    const suffix = `${testInfo.workerIndex}_${testInfo.repeatEachIndex}_${Date.now().toString(36)}`;
    const collection = `gyrifi_${suffix.replace(/[^a-zA-Z0-9_]/g, "_")}`;
    await startFreshStack(collection);
    try {
      await use({ collection });
    } finally {
      await stopStack(collection);
    }
  },
});

export { expect };

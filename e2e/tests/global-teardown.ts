import { stopStack } from "./harness";

export default async function globalTeardown(): Promise<void> {
  await stopStack("gyrifi_bootstrap");
}

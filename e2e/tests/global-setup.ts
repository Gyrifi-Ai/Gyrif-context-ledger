import { buildImage, startFreshStack } from "./harness";

export default async function globalSetup(): Promise<void> {
  await buildImage();
  await startFreshStack("gyrifi_bootstrap");
}

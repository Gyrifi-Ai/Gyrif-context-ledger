import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Providers } from "./providers";
import { Shell } from "./shell";

export function bootstrap(element: HTMLElement) {
  createRoot(element).render(<StrictMode><Providers><Shell /></Providers></StrictMode>);
}

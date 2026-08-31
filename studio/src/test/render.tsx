import type { ReactElement } from "react";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Providers } from "../app/providers";

export function renderWithProviders(ui: ReactElement, options?: Omit<RenderOptions, "wrapper">): RenderResult & { user: ReturnType<typeof userEvent.setup> } {
  const user = userEvent.setup();
  return { ...render(ui, { wrapper: Providers, ...options }), user };
}
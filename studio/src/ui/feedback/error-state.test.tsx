import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ErrorState } from "./error-state";

describe("ErrorState", () => {
  it("surfaces the server message verbatim and retries", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();
    render(<ErrorState title="Unable to load proposals" message="A release intent requires recovery." onRetry={retry} />);
    expect(screen.getByRole("alert")).toHaveTextContent("A release intent requires recovery.");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("supports caller-owned disabled actions and reasons", () => {
    render(<ErrorState message="Offline" onRetry={() => undefined} actionLabel="Reconnect" retryDisabled retryTitle="Wait for runtime" />);
    const action = screen.getByRole("button", { name: "Reconnect" });
    expect(action).toBeDisabled();
    expect(action).toHaveAttribute("title", "Wait for runtime");
  });
});

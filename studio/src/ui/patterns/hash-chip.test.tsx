import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { HashChip } from "./hash-chip";

describe("HashChip", () => {
  it("shows a labelled truncation while exposing the full value", () => {
    render(<HashChip label="HEAD" value="sha256:123456789abcdef" />);
    expect(screen.getByRole("button", { name: /HEAD · sha256:123/ })).toHaveAttribute("title", "sha256:123456789abcdef");
  });

  it("copies the complete value", async () => {
    const user = userEvent.setup();
    render(<HashChip value="sha256:123456789abcdef" />);
    await user.click(screen.getByRole("button"));
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(await navigator.clipboard.readText()).toBe("sha256:123456789abcdef");
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Textarea } from "./textarea";

describe("Textarea", () => {
  it("accepts multiline keyboard input", async () => {
    const user = userEvent.setup();
    render(<Textarea aria-label="Desired JSON" />);
    const input = screen.getByRole("textbox", { name: "Desired JSON" });
    await user.type(input, "{enter}");
    expect(input).toHaveValue("\n");
  });

  it("supports the disabled state", () => {
    render(<Textarea aria-label="Read-only evidence" disabled />);
    expect(screen.getByRole("textbox", { name: "Read-only evidence" })).toBeDisabled();
  });
});

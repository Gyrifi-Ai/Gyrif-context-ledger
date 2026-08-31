import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Input } from "./input";

describe("Input", () => {
  it("accepts keyboard input and forwards accessible attributes", async () => {
    const user = userEvent.setup();
    render(<Input aria-label="Unit" placeholder="point/1" />);
    const input = screen.getByRole("textbox", { name: "Unit" });
    await user.type(input, "point/42");
    expect(input).toHaveValue("point/42");
    expect(input).toHaveAttribute("placeholder", "point/1");
  });

  it("supports the disabled state", () => {
    render(<Input aria-label="Locked unit" disabled />);
    expect(screen.getByRole("textbox", { name: "Locked unit" })).toBeDisabled();
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { CodeBlock } from "./code-block";

describe("CodeBlock", () => {
  it("formats objects and labels the code language", () => {
    render(<CodeBlock value={{ ready: true }} language="json" />);
    expect(screen.getByLabelText("json")).toHaveTextContent('"ready": true');
  });

  it("copies string content", async () => {
    const user = userEvent.setup();
    render(<CodeBlock value="evidence" language="text" />);
    await user.click(screen.getByRole("button", { name: "Copy" }));
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(await navigator.clipboard.readText()).toBe("evidence");
  });
});

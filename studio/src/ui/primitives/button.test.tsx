import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CheckIcon } from "./icon";
import { Button } from "./button";

describe("Button", () => {
  it("renders every variant and size as an operable button", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<>{(["primary", "secondary", "ghost", "danger"] as const).map((variant) => <Button key={variant} variant={variant} size={variant === "primary" ? "md" : "sm"} onClick={onClick}>{variant}</Button>)}</>);
    for (const name of ["primary", "secondary", "ghost", "danger"]) await user.click(screen.getByRole("button", { name }));
    expect(onClick).toHaveBeenCalledTimes(4);
  });

  it("disables while loading and exposes busy state", () => {
    render(<Button loading>Save</Button>);
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toHaveAttribute("aria-busy", "true");
  });

  it("supports leading and accessible icon-only content", () => {
    render(<><Button iconLeft={<CheckIcon />}>Approve</Button><Button variant="ghost" aria-label="Copy hash"><CheckIcon /></Button></>);
    expect(screen.getByRole("button", { name: "Approve" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Copy hash" })).toBeEnabled();
  });
});

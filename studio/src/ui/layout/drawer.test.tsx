import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Drawer } from "./drawer";

function Harness() {
  const [open, setOpen] = useState(false);
  return <><button onClick={() => setOpen(true)}>Inspect</button><Drawer open={open} onClose={() => setOpen(false)} title="Intent detail" footer={<button>Retry verification</button>}><p>Plan operations</p></Drawer></>;
}

describe("Drawer", () => {
  it("opens with its title, content, and footer, then closes through its control", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const trigger = screen.getByRole("button", { name: "Inspect" });
    await user.click(trigger);
    expect(screen.getByRole("dialog", { name: "Intent detail" })).toBeInTheDocument();
    expect(screen.getByText("Plan operations")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry verification" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog", { name: "Intent detail" })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it("closes with Escape and restores focus", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const trigger = screen.getByRole("button", { name: "Inspect" });
    await user.click(trigger);
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Intent detail" })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });
});

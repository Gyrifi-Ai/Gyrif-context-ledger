import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ConfirmDialog } from "./confirm-dialog";

function Harness({ onConfirm = () => undefined, loading = false, disabled = false }: { onConfirm?: () => void; loading?: boolean; disabled?: boolean }) {
  const [open, setOpen] = useState(false);
  return <><button onClick={() => setOpen(true)}>Open confirmation</button><ConfirmDialog open={open} onClose={() => setOpen(false)} title="Confirm release" consequence="This changes the target" affectedCount={3} confirmLabel="Release" confirmLoading={loading} confirmDisabled={disabled} confirmTitle={disabled ? "Not permitted" : undefined} onConfirm={onConfirm} /></>;
}

describe("ConfirmDialog", () => {
  it("traps focus and confirms only through the destructive action", async () => {
    const user = userEvent.setup();
    const confirm = vi.fn();
    render(<Harness onConfirm={confirm} />);
    const trigger = screen.getByRole("button", { name: "Open confirmation" });
    await user.click(trigger);
    expect(screen.getByRole("dialog", { name: "Confirm release" })).toBeInTheDocument();
    expect(screen.getByText("3 affected units")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
    await user.tab({ shift: true });
    expect(screen.getByRole("button", { name: "Release" })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
    await user.click(screen.getByRole("button", { name: "Release" }));
    expect(confirm).toHaveBeenCalledTimes(1);
  });

  it("dismisses on Escape, restores trigger focus, and never confirms", async () => {
    const user = userEvent.setup();
    const confirm = vi.fn();
    render(<Harness onConfirm={confirm} />);
    const trigger = screen.getByRole("button", { name: "Open confirmation" });
    await user.click(trigger);
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Confirm release" })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
    expect(confirm).not.toHaveBeenCalled();
  });

  it("projects loading and disabled confirmation states", async () => {
    const user = userEvent.setup();
    render(<Harness loading disabled />);
    await user.click(screen.getByRole("button", { name: "Open confirmation" }));
    expect(screen.getByRole("button", { name: "Release" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Release" })).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("button", { name: "Release" })).toHaveAttribute("title", "Not permitted");
  });
});

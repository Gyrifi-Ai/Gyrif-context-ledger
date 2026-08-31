import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Segmented } from "./segmented";

describe("Segmented", () => {
  it("renders each option and changes through pointer input", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Segmented value="PUT" onChange={onChange} options={[{ value: "PUT", label: "Put" }, { value: "DELETE", label: "Delete" }]} />);
    expect(screen.getByRole("group")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(onChange).toHaveBeenCalledWith("DELETE");
  });

  it("supports keyboard activation", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Segmented value="PUT" onChange={onChange} options={[{ value: "PUT", label: "Put" }, { value: "DELETE", label: "Delete" }]} />);
    await user.tab();
    await user.keyboard("{Enter}");
    expect(onChange).toHaveBeenCalledWith("PUT");
  });
});

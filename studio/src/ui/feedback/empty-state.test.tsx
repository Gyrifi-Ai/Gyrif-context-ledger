import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EmptyState } from "./empty-state";

describe("EmptyState", () => {
  it("renders default content, icon, description, and action", () => {
    render(<EmptyState icon={<span aria-hidden="true">+</span>} title="No ledgers yet" description="Create the first governed namespace" action={<button>Create ledger</button>} />);
    expect(screen.getByText("No ledgers yet")).toBeInTheDocument();
    expect(screen.getByText("Create the first governed namespace")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create ledger" })).toBeEnabled();
  });

  it("renders its compact variant with only required content", () => {
    render(<EmptyState variant="compact" title="Nothing selected" />);
    expect(screen.getByText("Nothing selected")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});

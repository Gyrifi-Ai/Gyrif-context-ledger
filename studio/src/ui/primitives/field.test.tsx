import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Field, FieldGroup } from "./field";
import { Input } from "./input";

describe("Field", () => {
  it("associates its label and hint with the control", () => {
    render(<Field label="Ledger name" hint="Use a durable namespace"><Input /></Field>);
    const input = screen.getByRole("textbox", { name: "Ledger name" });
    expect(input).toHaveAccessibleDescription("Use a durable namespace");
    expect(input).toHaveAttribute("aria-invalid", "false");
  });

  it("replaces the hint with an accessible error", () => {
    render(<Field label="Ledger name" hint="Optional hint" error="Name is required"><Input /></Field>);
    const input = screen.getByRole("textbox", { name: "Ledger name" });
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input).toHaveAccessibleDescription("Name is required");
    expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
    expect(screen.queryByText("Optional hint")).not.toBeInTheDocument();
  });

  it("renders grouped fields", () => {
    render(<FieldGroup><Field label="One"><Input /></Field><Field label="Two"><Input /></Field></FieldGroup>);
    expect(screen.getAllByRole("textbox")).toHaveLength(2);
  });
});

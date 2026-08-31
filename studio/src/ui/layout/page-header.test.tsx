import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PageHeader } from "./page-header";

describe("PageHeader", () => {
  it("renders hierarchy and an optional action", () => {
    render(<PageHeader eyebrow="Durable inbox" title="Changes" description="Desired-state mutations" actions={<button>Submit change</button>} />);
    expect(screen.getByText("Durable inbox")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Changes" })).toBeInTheDocument();
    expect(screen.getByText("Desired-state mutations")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Submit change" })).toBeEnabled();
  });

  it("renders without actions", () => {
    render(<PageHeader eyebrow="History" title="Releases" description="Immutable history" />);
    expect(screen.getByRole("heading", { name: "Releases" })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});

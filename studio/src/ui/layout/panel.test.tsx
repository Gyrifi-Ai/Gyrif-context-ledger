import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Panel } from "./panel";

describe("Panel", () => {
  it("renders header metadata, actions, body, and footer", () => {
    render(<Panel eyebrow="Evidence" title="Latest evaluation" description="Bound to the current hash" actions={<button>Run again</button>} footer="Recorded"><p>Passed</p></Panel>);
    expect(screen.getByText("Evidence")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Latest evaluation" })).toBeInTheDocument();
    expect(screen.getByText("Bound to the current hash")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Run again" })).toBeEnabled();
    expect(screen.getByText("Passed")).toBeInTheDocument();
    expect(screen.getByText("Recorded")).toBeInTheDocument();
  });

  it("supports a flush body without a header", () => {
    render(<Panel padding="none"><table aria-label="Operations" /></Panel>);
    expect(screen.getByRole("table", { name: "Operations" })).toBeInTheDocument();
  });
});

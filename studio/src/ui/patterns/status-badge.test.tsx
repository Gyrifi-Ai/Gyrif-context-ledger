import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge, type StatusTone } from "./status-badge";

const tones = ["neutral", "info", "review", "success", "warning", "danger"] as const satisfies readonly StatusTone[];

describe("StatusBadge", () => {
  it("renders every tone with its explicit label", () => {
    render(<>{tones.map((tone) => <StatusBadge key={tone} label={tone} tone={tone} />)}</>);
    tones.forEach((tone) => expect(screen.getByText(tone)).toBeInTheDocument());
  });

  it("optionally renders a status dot without replacing the label", () => {
    const { container } = render(<StatusBadge label="READY" tone="neutral" dot />);
    expect(screen.getByText("READY")).toBeInTheDocument();
    expect(container.querySelectorAll("span span")).toHaveLength(1);
  });
});

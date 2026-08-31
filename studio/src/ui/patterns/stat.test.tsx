import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Stat } from "./stat";

describe("Stat", () => {
  it("renders values and every optional delta tone", () => {
    render(<>{(["default", "success", "warning", "danger"] as const).map((tone, index) => <Stat key={tone} label={`${tone} count`} value={index} delta={`${tone} delta`} tone={tone} />)}<Stat label="No delta" value="—" /></>);
    expect(screen.getByText("success delta")).toBeInTheDocument();
    expect(screen.getByText("warning delta")).toBeInTheDocument();
    expect(screen.getByText("danger delta")).toBeInTheDocument();
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

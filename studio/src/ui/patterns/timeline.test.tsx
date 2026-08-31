import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Timeline } from "./timeline";

describe("Timeline", () => {
  it("renders ordered items, current content, metadata, and bodies", () => {
    render(<Timeline items={[{ id: "head", node: <span>HEAD</span>, title: "Current release", meta: "Now", body: <button>View plan</button>, tone: "success", current: true }, { id: "old", title: "Older release", tone: "neutral" }]} />);
    expect(screen.getByRole("list")).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByText("HEAD")).toBeInTheDocument();
    expect(screen.getByText("Now")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "View plan" })).toBeEnabled();
  });
});

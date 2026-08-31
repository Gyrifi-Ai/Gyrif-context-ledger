import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Skeleton } from "./skeleton";

describe("Skeleton", () => {
  it("announces loading and renders the requested placeholder count", () => {
    const { container } = render(<Skeleton count={3} width="50%" height="2rem" radius="1rem" />);
    expect(container.querySelector("[aria-busy='true']")).toBeInTheDocument();
    expect(container.querySelectorAll("span")).toHaveLength(3);
  });

  it("defaults to one placeholder", () => {
    const { container } = render(<Skeleton />);
    expect(container.querySelectorAll("span")).toHaveLength(1);
  });
});

import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ApplicationShell } from "./application-shell";

describe("ApplicationShell", () => {
  it("renders navigation, topbar, banner, page header, content, and rail in landmarks", () => {
    render(<ApplicationShell sidebar={<nav>Navigation</nav>} topbar={<span>Runtime</span>} banner={<div role="alert">Offline</div>} header={<h1>Changes</h1>} rail={<section>Details</section>}><p>Inbox</p></ApplicationShell>);
    expect(screen.getByRole("navigation")).toHaveTextContent("Navigation");
    expect(screen.getByRole("alert")).toHaveTextContent("Offline");
    const main = screen.getByRole("main");
    expect(within(main).getByRole("heading", { name: "Changes" })).toBeInTheDocument();
    expect(within(main).getByText("Inbox")).toBeInTheDocument();
    expect(within(main).getByText("Details")).toBeInTheDocument();
  });

  it("renders without optional banner or rail", () => {
    render(<ApplicationShell sidebar="Navigation" topbar="Runtime" header="Header">Content</ApplicationShell>);
    expect(screen.getByRole("main")).toHaveTextContent("HeaderContent");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ApiError } from "../api/client";
import { mockApi } from "../test/api-mock";
import { renderWithProviders } from "../test/render";
import { ReachabilityBanner } from "./reachability-banner";
import { useReachability } from "./reachability";

function Health() {
  const { health } = useReachability();
  return <p>Health: {health.state}</p>;
}

describe("ReachabilityProvider", () => {
  it("moves from a transport outage to connected on explicit retry", async () => {
    mockApi.status.mockRejectedValueOnce(new Error("connection refused")).mockResolvedValue({ status: "ok", version: "1.2.3", commit: "abc123", buildDate: "2026-09-01T00:00:00Z", inference: "disabled" });
    const { user } = renderWithProviders(<><Health /><ReachabilityBanner /></>);
    expect(await screen.findByText("Health: offline")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Cannot reach the Gyrifi runtime");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("Health: connected")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("classifies an HTTP response as degraded rather than offline", async () => {
    mockApi.status.mockRejectedValue(new ApiError("UNAVAILABLE", "Starting", 503, "http"));
    renderWithProviders(<><Health /><ReachabilityBanner /></>);
    expect(await screen.findByText("Health: degraded")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});

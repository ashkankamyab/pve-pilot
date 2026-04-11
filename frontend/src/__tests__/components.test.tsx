import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import StatusBadge from "@/components/shared/StatusBadge";
import MetricBar from "@/components/shared/MetricBar";

describe("StatusBadge", () => {
  it("renders the status text", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("running")).toBeInTheDocument();
  });

  it("applies running style", () => {
    render(<StatusBadge status="running" />);
    const badge = screen.getByText("running");
    expect(badge.className).toContain("text-[#00ff88]");
  });

  it("applies stopped style", () => {
    render(<StatusBadge status="stopped" />);
    const badge = screen.getByText("stopped");
    expect(badge.className).toContain("text-red-400");
  });

  it("applies default style for unknown status", () => {
    render(<StatusBadge status="unknown" />);
    const badge = screen.getByText("unknown");
    expect(badge.className).toContain("text-[#888888]");
  });
});

describe("MetricBar", () => {
  it("renders label and values", () => {
    render(<MetricBar used={4} total={8} label="Memory" unit="GB" />);
    expect(screen.getByText("Memory")).toBeInTheDocument();
    expect(screen.getByText("4.0 GB / 8.0 GB")).toBeInTheDocument();
  });

  it("renders without unit", () => {
    render(<MetricBar used={2} total={4} label="CPU" />);
    expect(screen.getByText("CPU")).toBeInTheDocument();
    expect(screen.getByText("2.0 / 4.0")).toBeInTheDocument();
  });

  it("sets bar width based on percentage", () => {
    const { container } = render(
      <MetricBar used={3} total={10} label="Disk" unit="TB" />
    );
    const bar = container.querySelector("[style]");
    expect(bar).not.toBeNull();
    expect(bar!.getAttribute("style")).toContain("width: 30%");
  });

  it("caps bar width at 100%", () => {
    const { container } = render(
      <MetricBar used={15} total={10} label="Over" />
    );
    const bar = container.querySelector("[style]");
    expect(bar!.getAttribute("style")).toContain("width: 100%");
  });

  it("handles zero total gracefully", () => {
    const { container } = render(
      <MetricBar used={0} total={0} label="Empty" />
    );
    const bar = container.querySelector("[style]");
    expect(bar!.getAttribute("style")).toContain("width: 0%");
  });

  it("uses red color when usage > 80%", () => {
    const { container } = render(
      <MetricBar used={9} total={10} label="High" />
    );
    const bar = container.querySelector("[style]");
    expect(bar!.className).toContain("bg-red-500");
  });

  it("uses yellow color when usage > 60%", () => {
    const { container } = render(
      <MetricBar used={7} total={10} label="Medium" />
    );
    const bar = container.querySelector("[style]");
    expect(bar!.className).toContain("bg-yellow-500");
  });

  it("uses green color when usage <= 60%", () => {
    const { container } = render(
      <MetricBar used={3} total={10} label="Low" />
    );
    const bar = container.querySelector("[style]");
    expect(bar!.className).toContain("bg-[#00ff88]");
  });
});

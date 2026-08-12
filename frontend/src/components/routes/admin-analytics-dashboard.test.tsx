// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import type { AdminAnalyticsRange } from "#/lib/admin-analytics";

const { DateRangePicker, defaultDateRange } =
  await import("#/components/routes/admin-analytics-dashboard");

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-08-12T12:00:00Z"));
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("DateRangePicker", () => {
  it("defaults to the complete current month", () => {
    expect(defaultDateRange()).toEqual({ from: "2026-08-01", to: "2026-08-31" });
  });

  it("applies presets to the selected calendar range", () => {
    const onChange = vi.fn<(value: AdminAnalyticsRange) => void>();

    render(
      <DateRangePicker value={{ from: "2026-08-01", to: "2026-08-31" }} onChange={onChange} />,
    );

    fireEvent.click(screen.getByRole("button", { name: /Aug 1, 2026/ }));
    fireEvent.click(screen.getByRole("button", { name: "This week" }));

    expect(onChange).toHaveBeenCalledWith({ from: "2026-08-10", to: "2026-08-16" });
    expect(document.querySelector('[data-day="8/10/2026"]')?.getAttribute("data-range-start")).toBe(
      "true",
    );
  });

  it("centers the calendar in the wider presets popover", () => {
    render(<DateRangePicker value={{ from: "2026-08-01", to: "2026-08-31" }} onChange={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: /Aug 1, 2026/ }));

    expect(document.querySelector('[data-slot="calendar"]')?.className).toContain("mx-auto");
  });
});

import { describe, expect, it } from "vitest";
import {
  analyticsPeriodRange,
  currentAnalyticsDate,
  shiftAnalyticsPeriod,
  shiftCalendarDate,
} from "#/lib/admin-analytics-date";

describe("admin analytics calendar dates", () => {
  it("uses the Riga calendar day during standard time", () => {
    expect(currentAnalyticsDate(new Date("2026-01-31T22:30:00Z"))).toBe("2026-02-01");
  });

  it("uses the Riga calendar day during daylight saving time", () => {
    expect(currentAnalyticsDate(new Date("2026-07-31T21:30:00Z"))).toBe("2026-08-01");
  });

  it("shifts date-only values without depending on the runtime timezone", () => {
    expect(shiftCalendarDate("2026-03-01", -1)).toBe("2026-02-28");
    expect(shiftCalendarDate("2026-12-31", 1)).toBe("2027-01-01");
  });

  it("builds complete calendar week, month, and year periods", () => {
    expect(analyticsPeriodRange("week", "2026-07-31")).toEqual({
      from: "2026-07-27",
      to: "2026-08-02",
    });
    expect(analyticsPeriodRange("month", "2024-02-12")).toEqual({
      from: "2024-02-01",
      to: "2024-02-29",
    });
    expect(analyticsPeriodRange("year", "2026-07-31")).toEqual({
      from: "2026-01-01",
      to: "2026-12-31",
    });
  });

  it("moves preset ranges by their calendar period", () => {
    expect(shiftAnalyticsPeriod({ from: "2026-07-01", to: "2026-07-31" }, "month", 1)).toEqual({
      from: "2026-08-01",
      to: "2026-08-31",
    });
    expect(shiftAnalyticsPeriod({ from: "2026-12-28", to: "2027-01-03" }, "week", 1)).toEqual({
      from: "2027-01-04",
      to: "2027-01-10",
    });
    expect(shiftAnalyticsPeriod({ from: "2024-01-01", to: "2024-12-31" }, "year", 1)).toEqual({
      from: "2025-01-01",
      to: "2025-12-31",
    });
  });

  it("moves a custom range by its inclusive duration", () => {
    expect(shiftAnalyticsPeriod({ from: "2026-07-10", to: "2026-07-12" }, "custom", -1)).toEqual({
      from: "2026-07-07",
      to: "2026-07-09",
    });
  });
});

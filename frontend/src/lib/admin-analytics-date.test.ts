import { describe, expect, it } from "vitest";
import { currentAnalyticsDate, shiftCalendarDate } from "#/lib/admin-analytics-date";

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
});

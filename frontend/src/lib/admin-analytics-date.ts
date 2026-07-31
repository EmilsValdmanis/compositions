const ANALYTICS_TIME_ZONE = "Europe/Riga";
const DAY_IN_MILLISECONDS = 24 * 60 * 60 * 1000;

export type AnalyticsPeriod = "custom" | "week" | "month" | "year";

export type AnalyticsDateRange = {
  from: string;
  to: string;
};

const ANALYTICS_DATE_FORMATTER = new Intl.DateTimeFormat("en", {
  timeZone: ANALYTICS_TIME_ZONE,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

export function currentAnalyticsDate(reference = new Date()) {
  let year = "";
  let month = "";
  let day = "";

  for (const part of ANALYTICS_DATE_FORMATTER.formatToParts(reference)) {
    if (part.type === "year") year = part.value;
    if (part.type === "month") month = part.value;
    if (part.type === "day") day = part.value;
  }
  if (!year || !month || !day) throw new Error("failed to resolve the analytics date");

  return `${year}-${month}-${day}`;
}

export function shiftCalendarDate(value: string, days: number) {
  const shifted = parseCalendarDate(value);
  if (!Number.isInteger(days)) throw new Error("invalid calendar date shift");
  shifted.setUTCDate(shifted.getUTCDate() + days);
  return formatCalendarDate(shifted);
}

export function analyticsPeriodRange(
  period: Exclude<AnalyticsPeriod, "custom">,
  anchor: string,
): AnalyticsDateRange {
  const date = parseCalendarDate(anchor);
  const year = date.getUTCFullYear();
  const month = date.getUTCMonth();

  if (period === "week") {
    const daysSinceMonday = (date.getUTCDay() + 6) % 7;
    const from = shiftCalendarDate(anchor, -daysSinceMonday);
    return { from, to: shiftCalendarDate(from, 6) };
  }

  if (period === "month") {
    return {
      from: formatCalendarDate(new Date(Date.UTC(year, month, 1))),
      to: formatCalendarDate(new Date(Date.UTC(year, month + 1, 0))),
    };
  }

  return { from: `${year}-01-01`, to: `${year}-12-31` };
}

export function shiftAnalyticsPeriod(
  range: AnalyticsDateRange,
  period: AnalyticsPeriod,
  amount: number,
): AnalyticsDateRange {
  if (!Number.isInteger(amount)) throw new Error("invalid analytics period shift");

  if (period === "week") {
    return analyticsPeriodRange("week", shiftCalendarDate(range.from, amount * 7));
  }

  const from = parseCalendarDate(range.from);
  if (period === "month") {
    return analyticsPeriodRange(
      "month",
      formatCalendarDate(new Date(Date.UTC(from.getUTCFullYear(), from.getUTCMonth() + amount, 1))),
    );
  }

  if (period === "year") {
    return analyticsPeriodRange("year", `${from.getUTCFullYear() + amount}-01-01`);
  }

  const inclusiveDays = calendarDaysBetween(range.from, range.to) + 1;
  return {
    from: shiftCalendarDate(range.from, amount * inclusiveDays),
    to: shiftCalendarDate(range.to, amount * inclusiveDays),
  };
}

function parseCalendarDate(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) throw new Error("invalid calendar date");

  const [, yearValue, monthValue, dayValue] = match;
  const date = new Date(Date.UTC(Number(yearValue), Number(monthValue) - 1, Number(dayValue)));
  if (formatCalendarDate(date) !== value) throw new Error("invalid calendar date");
  return date;
}

function formatCalendarDate(value: Date) {
  return value.toISOString().slice(0, 10);
}

function calendarDaysBetween(from: string, to: string) {
  return Math.round(
    (parseCalendarDate(to).getTime() - parseCalendarDate(from).getTime()) / DAY_IN_MILLISECONDS,
  );
}

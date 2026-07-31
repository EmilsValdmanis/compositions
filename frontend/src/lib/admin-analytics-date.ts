const ANALYTICS_TIME_ZONE = "Europe/Riga";

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
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day || !Number.isInteger(days)) {
    throw new Error("invalid calendar date");
  }

  const shifted = new Date(Date.UTC(year, month - 1, day));
  shifted.setUTCDate(shifted.getUTCDate() + days);
  return shifted.toISOString().slice(0, 10);
}

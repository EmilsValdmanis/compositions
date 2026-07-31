import { queryOptions } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders, setResponseHeader } from "@tanstack/react-start/server";
import { z } from "zod";
import { authURL } from "#/lib/auth-shared";

const adminAnalyticsTotalsSchema = z.object({
  games: z.number().int().nonnegative(),
  activePlayers: z.number().int().nonnegative(),
  activePlaytimeSeconds: z.number().int().nonnegative(),
  healthyFinishRate: z.number().min(0).max(1),
  bugReports: z.number().int().nonnegative(),
  bugsResolved: z.number().int().nonnegative(),
  medianBugResolutionSeconds: z.number().nonnegative(),
});

const adminAnalyticsPointSchema = z.object({
  date: z.iso.date(),
  games: z.number().int().nonnegative(),
  activePlayers: z.number().int().nonnegative(),
  newPlayers: z.number().int().nonnegative(),
  returningPlayers: z.number().int().nonnegative(),
  completed: z.number().int().nonnegative(),
  forfeit: z.number().int().nonnegative(),
  mutualEnd: z.number().int().nonnegative(),
  technicalAbort: z.number().int().nonnegative(),
  abandoned: z.number().int().nonnegative(),
  inProgress: z.number().int().nonnegative(),
  bugReports: z.number().int().nonnegative(),
  bugsResolved: z.number().int().nonnegative(),
});

export const adminAnalyticsSchema = z.object({
  from: z.iso.date(),
  to: z.iso.date(),
  current: adminAnalyticsTotalsSchema,
  previous: adminAnalyticsTotalsSchema,
  points: z.array(adminAnalyticsPointSchema),
});
export type AdminAnalytics = z.infer<typeof adminAnalyticsSchema>;
export type AdminAnalyticsTotals = z.infer<typeof adminAnalyticsTotalsSchema>;

const adminAnalyticsRangeSchema = z.object({
  from: z.iso.date(),
  to: z.iso.date(),
});
export type AdminAnalyticsRange = z.infer<typeof adminAnalyticsRangeSchema>;

function adminRequestHeaders() {
  const requestHeaders = new Headers(getRequestHeaders());
  const headers = new Headers({ accept: "application/json" });
  const cookie = requestHeaders.get("cookie");
  if (cookie) headers.set("cookie", cookie);
  return headers;
}

export const getAdminAnalytics = createServerFn({ method: "GET" })
  .validator(adminAnalyticsRangeSchema)
  .handler(async ({ data }) => {
    setResponseHeader("cache-control", "private, no-store");
    const search = new URLSearchParams(data);
    const response = await fetch(
      authURL(`/api/admin/analytics?${search.toString()}`, process.env.VITE_GAME_SERVER_URL),
      { headers: adminRequestHeaders() },
    );
    if (!response.ok) throw new Error(`failed to load admin analytics: ${response.status}`);
    return adminAnalyticsSchema.parse(await response.json());
  });

export function adminAnalyticsOptions(dateRange: AdminAnalyticsRange) {
  return queryOptions({
    queryKey: ["admin", "analytics", dateRange.from, dateRange.to],
    queryFn: () => getAdminAnalytics({ data: dateRange }),
    staleTime: 60_000,
  });
}

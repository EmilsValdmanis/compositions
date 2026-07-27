import { queryOptions } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders, setResponseHeader } from "@tanstack/react-start/server";
import { z } from "zod";
import { authURL } from "#/lib/auth-shared";

export const ADMIN_BUG_REPORT_PAGE_SIZE = 20;

const bugReportBaseSchema = z.object({
  id: z.uuid(),
  roomCode: z.string().min(1),
  reporterPlayerId: z.string().min(1),
  description: z.string().min(1),
  round: z.number().int().positive(),
  turn: z.number().int().positive(),
  requestedAbort: z.boolean(),
  createdAt: z.iso.datetime({ offset: true }),
});

export const adminBugReportSummarySchema = bugReportBaseSchema;
export type AdminBugReportSummary = z.infer<typeof adminBugReportSummarySchema>;

export const adminBugReportPageSchema = z.object({
  reports: z.array(adminBugReportSummarySchema),
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
  totalItems: z.number().int().nonnegative(),
  totalPages: z.number().int().nonnegative(),
});
export type AdminBugReportPage = z.infer<typeof adminBugReportPageSchema>;

export const adminBugReportDetailSchema = bugReportBaseSchema.extend({
  reporterUserId: z.uuid().optional(),
  gameState: z.json(),
});
export type AdminBugReportDetail = z.infer<typeof adminBugReportDetailSchema>;

export const persistedCardSchema = z.object({
  rank: z.number().int().min(1).max(13).optional(),
  suit: z.number().int().min(0).max(3).optional(),
  isJoker: z.boolean().optional(),
});

export const persistedGameStateSchema = z.looseObject({
  version: z.number().int().positive(),
  gameMode: z.enum(["full", "quick"]).optional(),
  players: z.array(
    z.looseObject({
      id: z.string().min(1),
      hand: z.array(persistedCardSchema),
      totalPoints: z.number().int(),
      pointsGained: z.number().int(),
      unadjustedTotalPoints: z.number().int().optional(),
      hasOpened: z.boolean(),
      forfeited: z.boolean().optional(),
    }),
  ),
  activeCompositions: z.array(
    z.looseObject({
      variant: z.string(),
      cards: z.array(persistedCardSchema),
    }),
  ),
  drawPile: z.array(persistedCardSchema),
  discardPile: z.array(persistedCardSchema),
  maxPlayers: z.number().int().positive(),
  phase: z.number().int().min(0).max(3),
  round: z.number().int().positive(),
  dealerIndex: z.number().int().nonnegative(),
  turn: z.looseObject({
    number: z.number().int().positive(),
    playerIndex: z.number().int().nonnegative(),
    hasDrawn: z.boolean(),
    mustUseDiscardDraw: z.boolean(),
    discardDrawCard: persistedCardSchema.optional(),
  }),
  roundWinnerIndex: z.number().int(),
});
export type PersistedGameState = z.infer<typeof persistedGameStateSchema>;

function adminRequestHeaders() {
  const requestHeaders = new Headers(getRequestHeaders());
  const headers = new Headers({ accept: "application/json" });
  const cookie = requestHeaders.get("cookie");
  if (cookie) headers.set("cookie", cookie);
  return headers;
}

export const getAdminBugReports = createServerFn({ method: "GET" })
  .validator(
    z.object({
      page: z.number().int().positive(),
      pageSize: z.number().int().min(1).max(100).default(ADMIN_BUG_REPORT_PAGE_SIZE),
    }),
  )
  .handler(async ({ data }) => {
    setResponseHeader("cache-control", "private, no-store");
    const search = new URLSearchParams({
      page: String(data.page),
      pageSize: String(data.pageSize),
    });
    const response = await fetch(
      authURL(`/api/admin/bug-reports?${search.toString()}`, process.env.VITE_GAME_SERVER_URL),
      { headers: adminRequestHeaders() },
    );
    if (!response.ok) throw new Error(`failed to load admin bug reports: ${response.status}`);
    return adminBugReportPageSchema.parse(await response.json());
  });

export const getAdminBugReport = createServerFn({ method: "GET" })
  .validator(z.uuid())
  .handler(async ({ data: reportId }) => {
    setResponseHeader("cache-control", "private, no-store");
    const response = await fetch(
      authURL(
        `/api/admin/bug-reports/${encodeURIComponent(reportId)}`,
        process.env.VITE_GAME_SERVER_URL,
      ),
      { headers: adminRequestHeaders() },
    );
    if (!response.ok) throw new Error(`failed to load admin bug report: ${response.status}`);
    return adminBugReportDetailSchema.parse(await response.json());
  });

export const completeAdminBugReport = createServerFn({ method: "POST" })
  .validator(z.uuid())
  .handler(async ({ data: reportId }) => {
    const response = await fetch(
      authURL(
        `/api/admin/bug-reports/${encodeURIComponent(reportId)}/complete`,
        process.env.VITE_GAME_SERVER_URL,
      ),
      { method: "POST", headers: adminRequestHeaders() },
    );
    if (!response.ok) throw new Error(`failed to complete admin bug report: ${response.status}`);
    return reportId;
  });

export function adminBugReportPageOptions(page: number) {
  return queryOptions({
    queryKey: ["admin", "bug-reports", page],
    queryFn: () => getAdminBugReports({ data: { page, pageSize: ADMIN_BUG_REPORT_PAGE_SIZE } }),
    staleTime: 15_000,
  });
}

export function adminBugReportDetailOptions(reportId: string | null) {
  return queryOptions({
    queryKey: ["admin", "bug-report", reportId],
    queryFn: () => getAdminBugReport({ data: reportId! }),
    enabled: reportId !== null,
    staleTime: 60_000,
  });
}

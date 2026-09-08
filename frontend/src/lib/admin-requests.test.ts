import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";

vi.mock("@tanstack/react-start", () => ({
  createServerFn: () => ({
    validator: () => ({ handler: (handler: unknown) => handler }),
  }),
}));
vi.mock("@tanstack/react-start/server", () => ({
  getRequestHeaders: vi.fn(),
  setResponseHeader: vi.fn(),
}));

const { getRequestHeaders, setResponseHeader } = await import("@tanstack/react-start/server");
const { getAdminAnalytics } = await import("#/lib/admin-analytics");
const { getAdminBugReports, getAdminBugReport, completeAdminBugReport } =
  await import("#/lib/admin-bug-reports");

const reportId = "550e8400-e29b-41d4-a716-446655440000";
const totals = {
  games: 0,
  activePlayers: 0,
  activePlaytimeSeconds: 0,
  healthyFinishRate: 0,
  bugReports: 0,
  bugsResolved: 0,
  medianBugResolutionSeconds: 0,
};
const report = {
  id: reportId,
  roomCode: "ROOM",
  reporterPlayerId: "player",
  description: "Broken game",
  round: 1,
  turn: 1,
  requestedAbort: false,
  createdAt: "2026-09-08T10:00:00Z",
  gameState: {},
};
const requests = [
  {
    name: "analytics",
    method: "GET",
    path: "/api/admin/analytics?from=2026-09-01&to=2026-09-08",
    run: () => getAdminAnalytics({ data: { from: "2026-09-01", to: "2026-09-08" } }),
    response: {
      from: "2026-09-01",
      to: "2026-09-08",
      current: totals,
      previous: totals,
      points: [],
    },
  },
  {
    name: "report list",
    method: "GET",
    path: "/api/admin/bug-reports?page=2&pageSize=20",
    run: () => getAdminBugReports({ data: { page: 2, pageSize: 20 } }),
    response: { reports: [], page: 2, pageSize: 20, totalItems: 0, totalPages: 0 },
  },
  {
    name: "report detail",
    method: "GET",
    path: `/api/admin/bug-reports/${reportId}`,
    run: () => getAdminBugReport({ data: reportId }),
    response: report,
  },
  {
    name: "complete report",
    method: "POST",
    path: `/api/admin/bug-reports/${reportId}/complete`,
    run: () => completeAdminBugReport({ data: reportId }),
    response: reportId,
  },
];

describe("admin backend requests", () => {
  const fetchMock = vi.fn<typeof fetch>();
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubEnv("VITE_GAME_SERVER_URL", "https://game.example.com");
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  for (const request of requests) {
    it.each(["session=first", undefined, "session=second"])(
      `${request.name} forwards only the current cookie (%s)`,
      async (cookie) => {
        vi.mocked(getRequestHeaders).mockReturnValue(
          new Headers({
            ...(cookie ? { cookie } : {}),
            authorization: "do-not-forward",
            "x-unrelated": "private",
          }),
        );
        fetchMock.mockResolvedValue(Response.json(request.response));
        expect(await request.run()).toEqual(request.response);
        const [url, options] = fetchMock.mock.calls[0]!;
        expect(url).toBe(`https://game.example.com${request.path}`);
        expect(options?.method ?? "GET").toBe(request.method);
        expect(Object.fromEntries(new Headers(options?.headers))).toEqual({
          accept: "application/json",
          ...(cookie ? { cookie } : {}),
        });
        if (request.method === "GET") {
          expect(setResponseHeader).toHaveBeenCalledWith("cache-control", "private, no-store");
        }
      },
    );

    it(`${request.name} preserves backend errors`, async () => {
      vi.mocked(getRequestHeaders).mockReturnValue(new Headers());
      fetchMock.mockResolvedValue(new Response(null, { status: 403 }));
      await expect(request.run()).rejects.toThrow(/403/);
    });
  }
});

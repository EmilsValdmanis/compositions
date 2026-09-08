// @vitest-environment jsdom
import { type ReactNode } from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import { getLocale, overwriteGetLocale, type Locale } from "#/paraglide/runtime.js";
import { type PlayerProfile } from "#/lib/player-profile";

const { useInfiniteQuery, useQuery } = vi.hoisted(() => ({
  useInfiniteQuery: vi.fn(),
  useQuery: vi.fn(),
}));
vi.mock("@tanstack/react-query", () => ({
  useInfiniteQuery,
  useQuery,
  infiniteQueryOptions: (options: unknown) => options,
  keepPreviousData: (data: unknown) => data,
}));
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => <a>{children}</a>,
}));
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: () => ({
    getVirtualItems: () => [{ index: 0, key: "player", start: 0, end: 57 }],
    getTotalSize: () => 57,
    measureElement: () => {},
  }),
}));
vi.mock("#/lib/leaderboard", async (importOriginal) => ({
  ...(await importOriginal<typeof import("#/lib/leaderboard")>()),
  DEFAULT_LEADERBOARD_METRIC: "playtime",
}));

const { LeaderboardPage } = await import("#/components/routes/leaderboard-page");
const { PlayerProfilePage } = await import("#/components/routes/player-profile-page");
const originalGetLocale = getLocale;
afterEach(() => {
  cleanup();
  overwriteGetLocale(originalGetLocale);
});

const emptyStatistics: PlayerProfile["quick"] = {
  gamesPlayed: 0,
  gamesWon: 0,
  totalPlacement: 0,
  totalPlaytimeSeconds: 0,
  roundsPlayed: 0,
  roundsWon: 0,
  compositionsCreated: 0,
  setsCreated: 0,
  runsCreated: 0,
  pointsInflicted: 0,
  penaltyPoints: 0,
  currentGameWinStreak: 0,
  longestGameWinStreak: 0,
  currentRoundWinStreak: 0,
  longestRoundWinStreak: 0,
};
const cases = [
  { seconds: 0, en: "0m", lv: "0 min" },
  { seconds: 59, en: "0m", lv: "0 min" },
  { seconds: 60, en: "1m", lv: "1 min" },
  { seconds: 3599, en: "59m", lv: "59 min" },
  { seconds: 3600, en: "1h 0m", lv: "1 h 0 min" },
  { seconds: 7259, en: "2h 0m", lv: "2 h 0 min" },
];

describe("player-facing playtime", () => {
  for (const locale of ["en", "lv"] satisfies Locale[]) {
    it.each(cases)(
      `preserves truncated minutes on leaderboard and profile (${locale}, $seconds seconds)`,
      ({ seconds, ...labels }) => {
        overwriteGetLocale(() => locale);
        useInfiniteQuery.mockReturnValue({
          data: {
            pages: [
              {
                players: [
                  {
                    playerId: "player",
                    name: "Avery",
                    rank: 1,
                    score: seconds,
                  },
                ],
                placement: null,
              },
            ],
          },
        });
        const leaderboard = render(<LeaderboardPage playerId="player" />);
        expect(leaderboard.getByText(labels[locale]).tagName).toBe("TD");
        leaderboard.unmount();

        const history = { games: [], page: 1, pageSize: 10, totalItems: 0, totalPages: 0 };
        useQuery.mockReturnValue({ data: history });
        const profile = render(
          <PlayerProfilePage
            profile={{
              id: "player",
              name: "Avery",
              imageUrl: "",
              quick: emptyStatistics,
              rankedFull: { ...emptyStatistics, gamesPlayed: 1, totalPlaytimeSeconds: seconds },
            }}
            initialHistory={history}
          />,
        );
        expect(profile.getByText(labels[locale])).toBeTruthy();
      },
    );
  }
});

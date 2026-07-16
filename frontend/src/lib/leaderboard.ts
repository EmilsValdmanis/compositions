import { infiniteQueryOptions } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { z } from "zod";
import { authURL } from "#/lib/auth-shared";

export const LEADERBOARD_PAGE_SIZE = 50;

export const leaderboardPlayerSchema = z.object({
  rank: z.number().int().positive(),
  playerId: z.uuid(),
  name: z.string(),
  imageUrl: z.string(),
  wins: z.number().int().nonnegative(),
  gamesPlayed: z.number().int().positive(),
  roundsWon: z.number().int().nonnegative(),
  pointsInflicted: z.number().int().nonnegative(),
  totalPlaytimeSeconds: z.number().int().nonnegative(),
});

export type LeaderboardPlayer = z.infer<typeof leaderboardPlayerSchema>;

export const leaderboardPageSchema = z.object({
  players: z.array(leaderboardPlayerSchema),
  nextCursor: z.string().nullable(),
  placement: leaderboardPlayerSchema.nullable(),
});

export type LeaderboardPage = z.infer<typeof leaderboardPageSchema>;

export const getLeaderboardPage = createServerFn({ method: "GET" })
  .validator(
    z.object({
      cursor: z.string().nullable(),
      playerId: z.uuid(),
    }),
  )
  .handler(async ({ data }) => {
    const search = new URLSearchParams({
      limit: String(LEADERBOARD_PAGE_SIZE),
      playerId: data.playerId,
    });
    if (data.cursor) search.set("cursor", data.cursor);

    const response = await fetch(
      authURL(`/api/leaderboard?${search.toString()}`, process.env.VITE_GAME_SERVER_URL),
      { headers: { accept: "application/json" } },
    );
    if (!response.ok) throw new Error(`failed to load leaderboard: ${response.status}`);
    return leaderboardPageSchema.parse(await response.json());
  });

export function leaderboardInfiniteOptions(playerId: string) {
  return infiniteQueryOptions({
    queryKey: ["leaderboard", playerId],
    queryFn: ({ pageParam }) => getLeaderboardPage({ data: { cursor: pageParam, playerId } }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    staleTime: 30_000,
  });
}

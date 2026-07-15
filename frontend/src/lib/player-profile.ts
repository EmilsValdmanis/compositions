import { createServerFn } from "@tanstack/react-start";
import { z } from "zod";
import { authURL } from "#/lib/auth-shared";

export const playerProfileSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  imageUrl: z.string(),
  gamesPlayed: z.number(),
  gamesWon: z.number(),
  totalPlacement: z.number(),
  totalPlaytimeSeconds: z.number(),
  roundsPlayed: z.number(),
  roundsWon: z.number(),
  compositionsCreated: z.number(),
  setsCreated: z.number(),
  runsCreated: z.number(),
  pointsInflicted: z.number(),
  penaltyPoints: z.number(),
  currentGameWinStreak: z.number(),
  longestGameWinStreak: z.number(),
  currentRoundWinStreak: z.number(),
  longestRoundWinStreak: z.number(),
});

export type PlayerProfile = z.infer<typeof playerProfileSchema>;

export const playerGameHistoryItemSchema = z.object({
  id: z.uuid(),
  status: z.enum(["completed", "forfeit"]),
  completedAt: z.iso.datetime({ offset: true }),
  placement: z.number().int().positive(),
  playerCount: z.number().int().min(2).max(4),
  won: z.boolean(),
  forfeited: z.boolean(),
  totalPoints: z.number().int().nonnegative(),
  roundsPlayed: z.number().int().positive(),
  roundsWon: z.number().int().nonnegative(),
  playtimeSeconds: z.number().int().nonnegative(),
});

export const playerGameHistorySchema = z.object({
  games: z.array(playerGameHistoryItemSchema),
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
  totalItems: z.number().int().nonnegative(),
  totalPages: z.number().int().nonnegative(),
});

export type PlayerGameHistory = z.infer<typeof playerGameHistorySchema>;

export const getPlayerProfile = createServerFn({ method: "GET" })
  .validator(z.uuid())
  .handler(async ({ data: playerId }) => {
    const response = await fetch(
      authURL(`/api/players/${encodeURIComponent(playerId)}`, process.env.VITE_GAME_SERVER_URL),
      { headers: { accept: "application/json" } },
    );

    if (response.status === 404) return null;
    if (!response.ok) throw new Error(`failed to load player profile: ${response.status}`);
    return playerProfileSchema.parse(await response.json());
  });

export const getPlayerGameHistory = createServerFn({ method: "GET" })
  .validator(
    z.object({
      playerId: z.uuid(),
      page: z.number().int().positive(),
      pageSize: z.number().int().min(1).max(50).default(10),
    }),
  )
  .handler(async ({ data }) => {
    const path = `/api/players/${encodeURIComponent(data.playerId)}/games?page=${data.page}&pageSize=${data.pageSize}`;
    const response = await fetch(authURL(path, process.env.VITE_GAME_SERVER_URL), {
      headers: { accept: "application/json" },
    });

    if (!response.ok) throw new Error(`failed to load player game history: ${response.status}`);
    return playerGameHistorySchema.parse(await response.json());
  });

import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders } from "@tanstack/react-start/server";
import { z } from "zod";
import { authURL } from "#/lib/auth-shared";

export const playerProfileSchema = z.object({
  id: z.string().uuid(),
  name: z.string(),
  imageUrl: z.string(),
  gamesPlayed: z.number(),
  gamesWon: z.number(),
  totalPlacement: z.number(),
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

export const getPlayerProfile = createServerFn({ method: "GET" })
  .validator(z.string().uuid())
  .handler(async ({ data: playerId }) => {
    const requestHeaders = new Headers(getRequestHeaders());
    const cookie = requestHeaders.get("cookie");
    const response = await fetch(
      authURL(`/api/players/${encodeURIComponent(playerId)}`, process.env.VITE_GAME_SERVER_URL),
      {
        headers: cookie ? { cookie, accept: "application/json" } : { accept: "application/json" },
      },
    );

    if (response.status === 404) return null;
    if (!response.ok) throw new Error(`failed to load player profile: ${response.status}`);
    return playerProfileSchema.parse(await response.json());
  });

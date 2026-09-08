import { expect, it } from "vite-plus/test";
import {
  DEFAULT_LEADERBOARD_METRIC,
  leaderboardInfiniteOptions,
  leaderboardMetricSchema,
  leaderboardPageSchema,
} from "./leaderboard";

it("defaults to Elo and accepts all existing metrics", () => {
  expect(DEFAULT_LEADERBOARD_METRIC).toBe("elo");
  for (const metric of ["elo", "wins", "games", "playtime", "rounds", "points"]) {
    expect(leaderboardMetricSchema.parse(metric)).toBe(metric);
  }
  expect(leaderboardMetricSchema.safeParse("rating").success).toBe(false);
});

it("validates Elo scores and tiers in both players and placement", () => {
  const player = {
    rank: 1,
    score: 1000,
    tier: "bronze",
    playerId: "00000000-0000-4000-8000-000000000001",
    name: "Avery",
    imageUrl: "",
    wins: 1,
    gamesPlayed: 1,
    roundsWon: 1,
    pointsInflicted: 0,
    totalPlaytimeSeconds: 0,
  };
  const page = {
    reset: false,
    metric: "elo",
    scope: "friends",
    players: [player],
    placement: player,
    nextCursor: null,
  };
  expect(leaderboardPageSchema.parse(page)).toEqual(page);
  const futurePlayer = { ...player, tier: "champion" };
  expect(
    leaderboardPageSchema.parse({ ...page, players: [futurePlayer], placement: futurePlayer })
      .placement?.tier,
  ).toBe("champion");
  for (const invalid of [
    { ...player, score: -1 },
    { ...player, score: 1000.5 },
    { ...player, tier: "" },
  ]) {
    expect(leaderboardPageSchema.safeParse({ ...page, players: [invalid] }).success).toBe(false);
    expect(leaderboardPageSchema.safeParse({ ...page, placement: invalid }).success).toBe(false);
  }
});

it("isolates cached pages by metric and scope", () => {
  expect(leaderboardInfiniteOptions("viewer", "elo", "friends").queryKey).toEqual([
    "leaderboard",
    "viewer",
    "elo",
    "friends",
  ]);
  expect(leaderboardInfiniteOptions("viewer", "wins", "global").queryKey).toEqual([
    "leaderboard",
    "viewer",
    "wins",
    "global",
  ]);
});

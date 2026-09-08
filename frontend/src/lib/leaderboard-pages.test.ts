import { expect, it } from "vite-plus/test";
import type { LeaderboardPage, LeaderboardPlayer } from "./leaderboard";
import { visibleLeaderboard } from "./leaderboard-pages";

const player = (playerId: string, score = 1000): LeaderboardPlayer => ({
  playerId,
  score,
  rank: 1,
  name: playerId,
  imageUrl: "",
  wins: 1,
  gamesPlayed: 1,
  roundsWon: 1,
  pointsInflicted: 0,
  totalPlaytimeSeconds: 0,
});
const page = (
  players: LeaderboardPlayer[],
  reset = false,
  placement: LeaderboardPlayer | null = null,
): LeaderboardPage => ({
  players,
  reset,
  placement,
  metric: "elo",
  scope: "global",
  nextCursor: null,
});

it("replaces old pages and pinned placement when the server restarts pagination", () => {
  const oldA = player("A", 1016);
  const b = player("B");
  const c = player("C", 1001);
  const a = player("A", 999);
  const result = visibleLeaderboard([
    page([oldA, b], false, oldA),
    page([c, b], true, a),
    page([a]),
  ]);
  expect(result.players).toEqual([c, b, a]);
  expect(result.placement).toEqual(a);
  expect(result.resetIndex).toBe(1);
});

it("uses only the most recent restart, including an empty ranking", () => {
  expect(
    visibleLeaderboard([page([player("A")]), page([player("B")], true), page([], true)]),
  ).toEqual({ players: [], placement: null, resetIndex: 2 });
});

it("deduplicates player identities across pages", () => {
  const a = player("A");
  const updated = player("A", 999);
  const result = visibleLeaderboard([page([a]), page([updated, player("B")])]);
  expect(result.players).toEqual([updated, player("B")]);
});

it("handles an unloaded leaderboard", () => {
  expect(visibleLeaderboard([])).toEqual({ players: [], placement: null, resetIndex: -1 });
});

import type { LeaderboardPage } from "./leaderboard";

// A stale cursor returns a fresh first page. Never combine it with the ranking
// that preceded it, including when more pages have since been appended.
export function visibleLeaderboard(pages: LeaderboardPage[]) {
  const resetIndex = pages.findLastIndex((page) => page.reset);
  const currentPages = pages.slice(Math.max(0, resetIndex));
  const players = [
    ...new Map(
      currentPages.flatMap((page) => page.players).map((player) => [player.playerId, player]),
    ).values(),
  ];
  return { players, placement: currentPages[0]?.placement ?? null, resetIndex };
}

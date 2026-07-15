import { type GameSnapshot } from "#/components/game-websocket-provider";

export type ResultScorePhase = "previous" | "round" | "adjusted";

export function resultScoreState(
  player: Pick<
    GameSnapshot["players"][number],
    "pointsGained" | "totalPoints" | "unadjustedTotalPoints"
  >,
  phase: ResultScorePhase,
) {
  const unadjustedTotal = player.unadjustedTotalPoints ?? player.totalPoints;
  const previousTotal = Math.max(0, unadjustedTotal - player.pointsGained);
  const hasAdjustment = unadjustedTotal > 100 && unadjustedTotal !== player.totalPoints;

  return {
    displayedTotal:
      phase === "previous"
        ? previousTotal
        : phase === "round"
          ? unadjustedTotal
          : player.totalPoints,
    displayedGained: phase === "previous" ? 0 : player.pointsGained,
    hasAdjustment,
    isShowingOverHundred: hasAdjustment && phase === "round",
    isShowingAdjustment: hasAdjustment && phase === "adjusted",
    unadjustedTotal,
  };
}

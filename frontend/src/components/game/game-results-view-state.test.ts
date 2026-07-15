import { describe, expect, it } from "vite-plus/test";
import { resultScoreState } from "#/components/game/game-results-state";
import { playersForResults } from "#/components/game/game-view-helpers";
import { type RoomSnapshot } from "#/components/game-websocket-provider";

describe("resultScoreState", () => {
  const flyingScore = {
    pointsGained: 12,
    totalPoints: 89,
    unadjustedTotalPoints: 107,
  };

  it("reveals a flying score in previous, over-100, and adjusted phases", () => {
    expect(resultScoreState(flyingScore, "previous")).toMatchObject({
      displayedTotal: 95,
      displayedGained: 0,
      hasAdjustment: true,
    });
    expect(resultScoreState(flyingScore, "round")).toMatchObject({
      displayedTotal: 107,
      displayedGained: 12,
      isShowingOverHundred: true,
    });
    expect(resultScoreState(flyingScore, "adjusted")).toMatchObject({
      displayedTotal: 89,
      displayedGained: 12,
      isShowingAdjustment: true,
    });
  });

  it("keeps ordinary scores on the normal two-value reveal", () => {
    const ordinaryScore = { pointsGained: 9, totalPoints: 89, unadjustedTotalPoints: 89 };

    expect(resultScoreState(ordinaryScore, "previous").displayedTotal).toBe(80);
    expect(resultScoreState(ordinaryScore, "round").displayedTotal).toBe(89);
    expect(resultScoreState(ordinaryScore, "adjusted")).toMatchObject({
      displayedTotal: 89,
      hasAdjustment: false,
    });
  });
});

describe("playersForResults", () => {
  const resultRoom: RoomSnapshot = {
    code: "ROOM",
    phase: "round_over",
    hostPlayerId: "player-1",
    players: [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: true,
      },
    ],
  };

  it("uses live room players so result-screen emotes update", () => {
    const liveRoom: RoomSnapshot = {
      ...resultRoom,
      players: [
        {
          ...resultRoom.players[0]!,
          activeEmote: { id: "emote-1", emoji: "🎉", expiresAt: "2099-01-01T00:00:00Z" },
        },
      ],
    };

    expect(playersForResults(resultRoom, liveRoom)[0]?.activeEmote?.emoji).toBe("🎉");
  });

  it("keeps the result snapshot when the live room is unrelated", () => {
    expect(playersForResults(resultRoom, { ...resultRoom, code: "OTHER", players: [] })).toBe(
      resultRoom.players,
    );
  });
});

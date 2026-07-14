import { describe, expect, it } from "vite-plus/test";
import { inferCardTransfer } from "#/components/game/card-transfer-state";
import { type GameSnapshot } from "#/components/game-websocket-provider";

function game(overrides: Partial<GameSnapshot> = {}): GameSnapshot {
  return {
    phase: 1,
    round: 1,
    dealerIndex: 0,
    roundWinnerIndex: -1,
    turn: {
      number: 4,
      playerIndex: 1,
      playerId: "other-player",
      hasDrawn: false,
      mustUseDiscardDraw: false,
    },
    players: [
      {
        playerId: "viewer",
        handCount: 9,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: false,
      },
      {
        playerId: "other-player",
        handCount: 9,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: false,
      },
    ],
    hand: [],
    drawPileCount: 20,
    discardPile: [{ rank: 7, suit: 1 }],
    activeCompositions: [],
    ...overrides,
  };
}

describe("inferCardTransfer", () => {
  it("keeps deck draws face down while moving them to the active player", () => {
    const previous = game();
    const current = game({
      drawPileCount: 19,
      turn: { ...previous.turn, hasDrawn: true },
      players: previous.players.map((player) =>
        player.playerId === "other-player" ? { ...player, handCount: 10 } : player,
      ),
      turnActivity: {
        playerId: "other-player",
        round: 1,
        turnNumber: 4,
        drawSource: "deck",
      },
    });

    expect(inferCardTransfer(previous, current, "viewer")).toMatchObject({
      actorPlayerId: "other-player",
      faceDown: true,
      source: "deck",
      target: "player",
    });
  });

  it("uses the public top card for a discard-pile draw", () => {
    const previous = game();
    const current = game({
      discardPile: [],
      turn: { ...previous.turn, hasDrawn: true, mustUseDiscardDraw: true },
      players: previous.players.map((player) =>
        player.playerId === "other-player" ? { ...player, handCount: 10 } : player,
      ),
      turnActivity: {
        playerId: "other-player",
        round: 1,
        turnNumber: 4,
        drawSource: "discard",
      },
    });

    expect(inferCardTransfer(previous, current, "viewer")).toEqual({
      actorPlayerId: "other-player",
      card: { rank: 7, suit: 1 },
      faceDown: false,
      source: "discard",
      target: "player",
    });
  });

  it("moves a newly placed discard from the outgoing player to the pile", () => {
    const previous = game({
      turn: { ...game().turn, hasDrawn: true },
    });
    const current = game({
      turn: {
        ...previous.turn,
        number: 5,
        playerIndex: 0,
        playerId: "viewer",
        hasDrawn: false,
      },
      discardPile: [{ rank: 12, suit: 3 }, ...previous.discardPile],
    });

    expect(inferCardTransfer(previous, current, "viewer")).toEqual({
      actorPlayerId: "other-player",
      card: { rank: 12, suit: 3 },
      faceDown: false,
      source: "player",
      target: "discard",
    });
  });

  it("does not duplicate the current viewer's direct manipulation", () => {
    const previous = game();
    const current = game({
      drawPileCount: 19,
      turn: { ...previous.turn, hasDrawn: true },
      players: previous.players.map((player) =>
        player.playerId === "other-player" ? { ...player, handCount: 10 } : player,
      ),
    });

    expect(inferCardTransfer(previous, current, "other-player")).toBeNull();
  });
});

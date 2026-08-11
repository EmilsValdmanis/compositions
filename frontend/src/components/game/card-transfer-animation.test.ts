import { describe, expect, it } from "vite-plus/test";
import {
  buildCardTransferKeyframes,
  CARD_TRANSFER_DURATION_MS,
  CARD_TRANSFER_PLAYER_SCALE,
} from "#/components/game/card-transfer-animation";
import {
  inferCardTransfer,
  inferCompletedCompositionCollection,
} from "#/components/game/card-transfer-state";
import { buildCompletedCardCollectionKeyframes } from "#/components/game/completed-composition-animation";
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

describe("card transfer motion", () => {
  it("travels 50 percent slower than the original transfer", () => {
    expect(CARD_TRANSFER_DURATION_MS).toBe(420);
  });

  it("shrinks from pile size to card-icon size when drawing", () => {
    const frames = buildCardTransferKeyframes({
      source: "deck",
      target: "player",
      translateX: 100,
      translateY: 50,
    });

    expect(frames[0]?.transform).toContain("scale(1)");
    expect(frames.at(-1)?.transform).toContain(`scale(${CARD_TRANSFER_PLAYER_SCALE})`);
  });

  it("grows from card-icon size to pile size when discarding", () => {
    const frames = buildCardTransferKeyframes({
      source: "player",
      target: "discard",
      translateX: 100,
      translateY: 50,
    });

    expect(frames[0]?.transform).toContain(`scale(${CARD_TRANSFER_PLAYER_SCALE})`);
    expect(frames.at(-1)?.transform).toContain("scale(1)");
  });
});

describe("completed composition collection", () => {
  it("separates collected cards from the discard placed above them", () => {
    const previous = game({
      turn: { ...game().turn, hasDrawn: true },
      discardPile: [{ rank: 7, suit: 1 }],
    });
    const completedSet = [
      { rank: 9, suit: 0 },
      { rank: 9, suit: 1 },
      { rank: 9, suit: 2 },
      { rank: 9, suit: 3 },
    ];
    const current = game({
      discardPile: [{ rank: 12, suit: 3 }, ...completedSet, ...previous.discardPile],
    });

    expect(inferCompletedCompositionCollection(previous, current)).toEqual({
      actorPlayerId: "other-player",
      collectedCards: completedSet,
      discardCard: { rank: 12, suit: 3 },
      previousTopDiscard: { rank: 7, suit: 1 },
    });
  });

  it("ignores an ordinary one-card discard", () => {
    const previous = game();
    const current = game({
      discardPile: [{ rank: 12, suit: 3 }, ...previous.discardPile],
    });

    expect(inferCompletedCompositionCollection(previous, current)).toBeNull();
  });

  it("lifts before converging on the discard pile", () => {
    const frames = buildCompletedCardCollectionKeyframes(
      { left: 10, top: 20, width: 56, height: 80 },
      { left: 210, top: 120, width: 56, height: 80 },
    );

    expect(frames[1]?.transform).toContain("-16px");
    expect(frames.at(-1)?.transform).toContain("translate3d(200px, 100px, 0)");
  });
});

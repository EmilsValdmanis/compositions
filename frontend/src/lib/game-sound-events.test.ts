import { describe, expect, it } from "vite-plus/test";
import {
  type GameSnapshot,
  type LobbyState,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { gameSoundsForStateChange } from "#/lib/game-sound-events";

function makeGame(overrides: Partial<GameSnapshot> = {}): GameSnapshot {
  return {
    phase: 1,
    round: 1,
    dealerIndex: 0,
    roundWinnerIndex: -1,
    turn: {
      number: 1,
      playerIndex: 0,
      playerId: "player-1",
      hasDrawn: false,
      mustUseDiscardDraw: false,
    },
    players: [],
    hand: [],
    drawPileCount: 50,
    discardPile: [],
    activeCompositions: [],
    ...overrides,
  };
}

function makeRoom(overrides: Partial<RoomSnapshot> = {}): RoomSnapshot {
  return {
    code: "ROOM",
    phase: "in_progress",
    hostPlayerId: "player-1",
    players: [
      {
        playerId: "player-1",
        name: "One",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: true,
      },
    ],
    ...overrides,
  };
}

function makeState(overrides: Partial<LobbyState> = {}): LobbyState {
  return {
    connectionStatus: "connected",
    sessionId: "session-1",
    playerId: "player-1",
    room: makeRoom(),
    game: makeGame(),
    lastActionResult: null,
    lastError: null,
    lastErrorCode: null,
    lastErrorMessage: null,
    lastErrorId: 0,
    lastEvent: "game_state",
    completedGame: null,
    isSpectating: false,
    social: {
      userId: "",
      friends: [],
      incomingFriendRequests: [],
      outgoingFriendRequestUserIds: [],
      gameInvites: [],
    },
    ...overrides,
  };
}

describe("gameSoundsForStateChange", () => {
  it("announces lobby membership changes to the room", () => {
    const previous = makeState();
    const joinedPlayer = {
      playerId: "player-2",
      name: "Two",
      connected: true,
      seat: 1,
      isHost: false,
      canReconnect: true,
    };
    const joined = makeState({
      room: makeRoom({ players: [...previous.room!.players, joinedPlayer] }),
    });

    expect(gameSoundsForStateChange(previous, joined)).toContain("player-joined");
    expect(gameSoundsForStateChange(joined, previous)).toContain("player-left");
  });

  it("plays shared draw and discard cues and only alerts the active player", () => {
    const previous = makeState();
    const drawn = makeState({
      game: makeGame({ turn: { ...previous.game!.turn, hasDrawn: true } }),
    });
    const nextTurn = makeState({
      game: makeGame({
        turn: {
          ...previous.game!.turn,
          number: 2,
          hasDrawn: false,
          playerId: "player-1",
        },
      }),
    });

    expect(gameSoundsForStateChange(previous, drawn)).toEqual(["card-draw"]);
    expect(gameSoundsForStateChange(drawn, nextTurn)).toEqual(["card-discard", "turn-start"]);
    expect(
      gameSoundsForStateChange(drawn, {
        ...nextTurn,
        playerId: "player-2",
      }),
    ).toEqual(["card-discard"]);
  });

  it("distinguishes committed compositions and joker reclaims", () => {
    const previous = makeState();
    const composition = {
      type: "set",
      cards: [{ rank: 7, suit: 0 }],
      points: 7,
      complete: true,
    };
    const created = makeState({
      game: makeGame({
        activeCompositions: [composition],
        turnActivity: {
          playerId: "player-1",
          round: 1,
          turnNumber: 1,
          compositionActivities: [{ tableIndex: 0, kind: "new_composition" }],
        },
      }),
    });
    const reclaimed = makeState({
      game: makeGame({
        activeCompositions: [composition],
        turnActivity: {
          playerId: "player-1",
          round: 1,
          turnNumber: 1,
          compositionActivities: [
            {
              tableIndex: 0,
              cardActivities: {
                0: { kind: "joker_reclaim", playerId: "player-1" },
              },
            },
          ],
        },
      }),
    });

    expect(gameSoundsForStateChange(previous, created)).toContain("composition-create");
    expect(gameSoundsForStateChange(created, reclaimed)).toContain("joker-reclaim");
  });

  it("uses distinct round, game, and invalid-action cues", () => {
    const previous = makeState();

    expect(
      gameSoundsForStateChange(previous, makeState({ room: makeRoom({ phase: "round_over" }) })),
    ).toContain("round-win");
    expect(
      gameSoundsForStateChange(previous, makeState({ room: makeRoom({ phase: "game_over" }) })),
    ).toContain("game-win");
    expect(
      gameSoundsForStateChange(previous, {
        ...previous,
        lastError: "not your turn",
        lastErrorId: 1,
      }),
    ).toContain("invalid-action");
  });
});

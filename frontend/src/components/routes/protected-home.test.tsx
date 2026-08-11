// @vitest-environment jsdom

import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  COMPLETED_COLLECTION_DURATION_MS,
  COMPLETED_DISCARD_DURATION_MS,
} from "#/components/game/completed-composition-animation";
import { type GameSnapshot, type RoomSnapshot } from "#/components/game-websocket-provider";

const testRuntime = vi.hoisted(() => ({
  boardMountCount: 0,
  navigate: vi.fn(),
  useGameWebSocket: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  ClientOnly: ({ children }: { children: React.ReactNode }) => children,
  getRouteApi: () => ({
    useNavigate: () => testRuntime.navigate,
    useSearch: () => ({}),
  }),
}));

vi.mock("#/components/game-websocket-provider", () => ({
  useGameWebSocket: testRuntime.useGameWebSocket,
}));

vi.mock("#/components/game/game-board-view", async () => {
  const { useState } = await import("react");

  return {
    GameBoardView: () => {
      const [instanceId] = useState(() => {
        testRuntime.boardMountCount += 1;
        return testRuntime.boardMountCount;
      });

      return <div data-testid="game-board" data-instance-id={instanceId} />;
    },
  };
});

vi.mock("#/components/game/game-lobby-view", () => ({
  GameLobbyView: () => <div data-testid="game-lobby" />,
}));

vi.mock("#/components/game/game-results-view", () => ({
  GameResultsView: () => <div data-testid="game-results" />,
}));

vi.mock("#/components/game/end-game-proposal-alert", () => ({
  EndGameProposalAlert: () => null,
}));

vi.mock("#/components/routes/game-route-loading-screen", () => ({
  GameRouteLoadingScreen: () => <div data-testid="game-loading" />,
}));

vi.mock("#/lib/game-sound-events", () => ({
  useGameSoundEvents: () => undefined,
}));

const { ProtectedHome } = await import("#/components/routes/protected-home");

function game(overrides: Partial<GameSnapshot> = {}): GameSnapshot {
  return {
    phase: 1,
    round: 1,
    dealerIndex: 0,
    roundWinnerIndex: -1,
    turn: {
      number: 4,
      playerIndex: 0,
      playerId: "player-1",
      hasDrawn: true,
      mustUseDiscardDraw: false,
    },
    players: [
      {
        playerId: "player-1",
        handCount: 1,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: true,
      },
      {
        playerId: "player-2",
        handCount: 5,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: true,
      },
    ],
    hand: [{ rank: 13, suit: 3 }],
    drawPileCount: 20,
    discardPile: [{ rank: 3, suit: 3 }],
    activeCompositions: [
      {
        type: "set",
        cards: [
          { rank: 9, suit: 0 },
          { rank: 9, suit: 1 },
          { rank: 9, suit: 2 },
          { rank: 9, suit: 3 },
        ],
        points: 36,
        complete: true,
      },
    ],
    ...overrides,
  };
}

function room(phase: string): RoomSnapshot {
  return {
    code: "ROOM",
    phase,
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
      {
        playerId: "player-2",
        name: "Blair",
        connected: true,
        seat: 1,
        isHost: false,
        canReconnect: true,
      },
    ],
  };
}

function websocketValue(state: Record<string, unknown>) {
  const action = vi.fn();

  return {
    state,
    createRoom: action,
    joinRoom: action,
    leaveRoom: action,
    startGame: action,
    startNextRound: action,
    dismissCompletedGame: action,
    chooseDealing: action,
    sendEmote: action,
    sendFriendRequest: vi.fn().mockResolvedValue(undefined),
    stopSpectating: vi.fn().mockResolvedValue(undefined),
    drawFromDeck: action,
    drawFromDiscard: action,
    playTable: vi.fn().mockResolvedValue(undefined),
    playTableAndDiscard: vi.fn().mockResolvedValue(undefined),
    discardCard: vi.fn().mockResolvedValue(undefined),
  };
}

function lobbyState(currentRoom: RoomSnapshot, currentGame: GameSnapshot) {
  return {
    connectionStatus: "connected",
    sessionId: "session-1",
    playerId: "player-1",
    room: currentRoom,
    game: currentGame,
    lastActionResult: null,
    lastError: null,
    lastErrorCode: null,
    lastErrorMessage: null,
    lastErrorId: 0,
    lastEvent: null,
    completedGame: null,
    isSpectating: false,
    social: {
      userId: "user-1",
      friends: [],
      incomingFriendRequests: [],
      outgoingFriendRequestUserIds: [],
      gameInvites: [],
    },
  };
}

beforeEach(() => {
  vi.useFakeTimers();
  testRuntime.boardMountCount = 0;
  testRuntime.navigate.mockReset();
  testRuntime.useGameWebSocket.mockReset();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("ProtectedHome completed composition handoff", () => {
  it("keeps the existing board mounted while deferring round results", () => {
    const boardGame = game();
    testRuntime.useGameWebSocket.mockReturnValue(
      websocketValue(lobbyState(room("in_progress"), boardGame)),
    );
    const view = render(<ProtectedHome />);

    expect(view.getByTestId("game-board").getAttribute("data-instance-id")).toBe("1");

    const completedSet = boardGame.activeCompositions[0]?.cards ?? [];
    const roundOverGame = game({
      hand: [],
      activeCompositions: [],
      discardPile: [{ rank: 13, suit: 3 }, ...completedSet, ...boardGame.discardPile],
      roundWinnerIndex: 0,
    });
    testRuntime.useGameWebSocket.mockReturnValue(
      websocketValue(lobbyState(room("round_over"), roundOverGame)),
    );
    view.rerender(<ProtectedHome />);

    expect(view.queryByTestId("game-results")).toBeNull();
    expect(view.getByTestId("game-board").getAttribute("data-instance-id")).toBe("1");
    expect(testRuntime.boardMountCount).toBe(1);

    act(() => {
      vi.advanceTimersByTime(COMPLETED_COLLECTION_DURATION_MS + COMPLETED_DISCARD_DURATION_MS + 40);
    });

    expect(view.getByTestId("game-results")).toBeTruthy();
  });
});

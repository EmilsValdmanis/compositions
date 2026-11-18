// @vitest-environment jsdom

import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { type GameSnapshot, type RoomSnapshot } from "#/lib/game-protocol";

const testRuntime = vi.hoisted(() => ({
  boardMountCount: 0,
  settleBoardCollection: vi.fn<(game: GameSnapshot) => void>(),
  navigate: vi.fn(),
  toastError: vi.fn(),
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

vi.mock("sonner", () => ({
  toast: {
    error: testRuntime.toastError,
    success: vi.fn(),
  },
}));

vi.mock("#/components/game/game-board-view", async () => {
  const { useState } = await import("react");

  return {
    GameBoardView: ({
      game,
      onCollectionSettled,
      viewerMode,
    }: {
      game: GameSnapshot | null;
      onCollectionSettled: (game: GameSnapshot) => void;
      viewerMode: "player" | "spectator";
    }) => {
      testRuntime.settleBoardCollection.mockImplementation(onCollectionSettled);
      const [instanceId] = useState(() => {
        testRuntime.boardMountCount += 1;
        return testRuntime.boardMountCount;
      });

      return (
        <div
          data-testid="game-board"
          data-instance-id={instanceId}
          data-viewer-mode={viewerMode}
          data-game-phase={game?.phase}
        />
      );
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
  const dismissError = vi.fn();

  return {
    state,
    createRoom: action,
    joinRoom: action,
    leaveRoom: action,
    dismissError,
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

function lobbyState(currentRoom: RoomSnapshot | null, currentGame: GameSnapshot | null) {
  return {
    connectionStatus: "connected",
    sessionId: "session-1",
    playerId: "player-1",
    room: currentRoom,
    game: currentGame,
    lastError: null,
    lastErrorCode: null,
    lastErrorId: 0,
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
  testRuntime.settleBoardCollection.mockReset();
  testRuntime.boardMountCount = 0;
  testRuntime.navigate.mockReset();
  testRuntime.toastError.mockReset();
  testRuntime.useGameWebSocket.mockReset();
});

afterEach(() => {
  cleanup();
});

function gameWithCollectionFinished(previous: GameSnapshot, phase: number) {
  return game({
    phase,
    hand: [],
    activeCompositions: [],
    discardPile: [
      { rank: 13, suit: 3 },
      ...previous.activeCompositions.flatMap((composition) => composition.cards),
      ...previous.discardPile,
    ],
    roundWinnerIndex: 0,
  });
}

describe("ProtectedHome completed composition handoff", () => {
  it("keeps the board until the final round snapshot and its collection settle", () => {
    const boardGame = game();
    testRuntime.useGameWebSocket.mockReturnValue(
      websocketValue(lobbyState(room("in_progress"), boardGame)),
    );
    const view = render(<ProtectedHome />);

    testRuntime.useGameWebSocket.mockReturnValue(
      websocketValue(lobbyState(room("round_over"), boardGame)),
    );
    view.rerender(<ProtectedHome />);

    expect(view.queryByTestId("game-results")).toBeNull();
    expect(view.getByTestId("game-board").getAttribute("data-instance-id")).toBe("1");

    const roundOverGame = gameWithCollectionFinished(boardGame, 2);
    testRuntime.useGameWebSocket.mockReturnValue(
      websocketValue(lobbyState(room("round_over"), roundOverGame)),
    );
    view.rerender(<ProtectedHome />);

    expect(view.queryByTestId("game-results")).toBeNull();
    expect(view.getByTestId("game-board").getAttribute("data-instance-id")).toBe("1");

    act(() => {
      testRuntime.settleBoardCollection(roundOverGame);
    });

    expect(view.getByTestId("game-results")).toBeTruthy();
  });

  it.each(["player", "spectator"] as const)(
    "retains the terminal board snapshot for a %s after the live game is cleared",
    (viewerMode) => {
      const isSpectator = viewerMode === "spectator";
      const boardGame = game();
      testRuntime.useGameWebSocket.mockReturnValue(
        websocketValue({
          ...lobbyState(room("in_progress"), boardGame),
          isSpectating: isSpectator,
        }),
      );
      const view = render(<ProtectedHome />);
      const finalGame = gameWithCollectionFinished(boardGame, 3);
      const finalRoom = room("game_over");

      if (!isSpectator) {
        testRuntime.useGameWebSocket.mockReturnValue(
          websocketValue(lobbyState(finalRoom, finalGame)),
        );
        view.rerender(<ProtectedHome />);
      }

      testRuntime.useGameWebSocket.mockReturnValue(
        websocketValue({
          ...lobbyState(isSpectator ? null : room("lobby"), null),
          completedGame: { room: finalRoom, game: finalGame },
        }),
      );
      view.rerender(<ProtectedHome />);

      const board = view.getByTestId("game-board");
      expect(board.getAttribute("data-instance-id")).toBe("1");
      expect(board.getAttribute("data-game-phase")).toBe("3");
      expect(board.getAttribute("data-viewer-mode")).toBe(viewerMode);
      expect(view.queryByTestId("game-results")).toBeNull();

      act(() => {
        testRuntime.settleBoardCollection(finalGame);
      });

      expect(view.getByTestId("game-results")).toBeTruthy();
    },
  );
});

describe("ProtectedHome errors", () => {
  it("acknowledges an error after showing its toast", async () => {
    const state = {
      ...lobbyState(null, null),
      lastError: "The room was not found.",
      lastErrorCode: "room_not_found",
      lastErrorId: 1,
    };
    const connection = websocketValue(state);
    testRuntime.useGameWebSocket.mockReturnValue(connection);

    render(<ProtectedHome />);

    expect(testRuntime.toastError).toHaveBeenCalledWith(state.lastError);
    expect(connection.dismissError).toHaveBeenCalledOnce();
  });
});

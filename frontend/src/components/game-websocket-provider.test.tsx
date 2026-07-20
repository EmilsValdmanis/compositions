// @vitest-environment jsdom
import { createRequire } from "node:module";
import { type ReactNode } from "react";

if (typeof globalThis.document === "undefined") {
  const require = createRequire(import.meta.url);
  const { JSDOM } = require("jsdom") as {
    JSDOM: new (
      html?: string,
      options?: Record<string, unknown>,
    ) => {
      window: Window & typeof globalThis;
    };
  };

  const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    url: "http://localhost/",
  });

  globalThis.window = dom.window;
  globalThis.document = dom.window.document;
  globalThis.HTMLElement = dom.window.HTMLElement;
  globalThis.Node = dom.window.Node;
  globalThis.MutationObserver = dom.window.MutationObserver;
  globalThis.getComputedStyle = dom.window.getComputedStyle.bind(dom.window);
  Object.defineProperty(globalThis, "navigator", {
    value: dom.window.navigator,
    configurable: true,
  });
}

const { act, cleanup, fireEvent, render, screen, waitFor } = await import("@testing-library/react");
const { afterEach, beforeEach, describe, expect, it, vi } = await import("vite-plus/test");

vi.mock("#/lib/game-auth", () => ({
  getGameConnectionAuth: vi.fn().mockResolvedValue({ playerId: "player-1" }),
}));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => vi.fn(),
}));

const { GameWebSocketProvider, useGameWebSocket } =
  await import("#/components/game-websocket-provider");
const { NotificationsDropdown } = await import("#/components/social/notifications-dropdown");

const sockets: FakeWebSocket[] = [];

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  readonly url: string;
  readyState = FakeWebSocket.CONNECTING;
  sent: string[] = [];
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    sockets.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  message(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  close() {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({} as CloseEvent);
  }
}

function Harness({ children }: { children?: ReactNode }) {
  const { state, connect, disconnect, discardCard, playTableAndDiscard, updateTurnDrafts } =
    useGameWebSocket();

  return (
    <div>
      <p data-testid="session-id">{state.sessionId}</p>
      <p data-testid="connection-status">{state.connectionStatus}</p>
      <p data-testid="last-error">{state.lastError}</p>
      <p data-testid="last-error-code">{state.lastErrorCode}</p>
      <p data-testid="last-error-message">{state.lastErrorMessage}</p>
      <p data-testid="completed-game-phase">{state.completedGame?.room.phase}</p>
      <p data-testid="social-user-id">{state.social.userId}</p>
      <p data-testid="social-notification-count">
        {state.social.incomingFriendRequests.length + state.social.gameInvites.length}
      </p>
      <button type="button" onClick={() => void connect()}>
        Connect
      </button>
      <button type="button" onClick={disconnect}>
        Disconnect
      </button>
      <button type="button" onClick={() => void discardCard(4, { rank: 12, suit: 2 })}>
        Discard test card
      </button>
      <button type="button" onClick={() => updateTurnDrafts({ compositions: [] })}>
        Sync drafts
      </button>
      <button
        type="button"
        onClick={() =>
          void playTableAndDiscard({ compositions: [], additions: [], reclaims: [] }, 1, {
            isJoker: true,
          })
        }
      >
        Play and discard test card
      </button>
      {children}
    </div>
  );
}

function connectEnvelope(socket: FakeWebSocket) {
  return JSON.parse(socket.sent[0] ?? "{}") as {
    type?: string;
    data?: { sessionId?: string };
  };
}

describe("GameWebSocketProvider", () => {
  beforeEach(() => {
    cleanup();
    sockets.splice(0);
    window.localStorage.clear();
    process.env.VITE_GAME_SERVER_URL = "ws://localhost:8080";
    Object.defineProperty(globalThis, "WebSocket", {
      value: FakeWebSocket,
      configurable: true,
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("restores the server session id when reconnecting", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));

    await act(async () => {
      sockets[0]!.open();
    });
    expect(connectEnvelope(sockets[0]!)).toEqual({
      type: "connect",
      data: { sessionId: "" },
    });

    await act(async () => {
      sockets[0]!.message({
        type: "connected",
        data: { sessionId: "session-1", playerId: "player-1" },
      });
    });
    await waitFor(() => expect(screen.getByTestId("session-id").textContent).toBe("session-1"));
    expect(window.localStorage.getItem("compositions.game.sessionId")).toBeNull();

    await act(async () => {
      sockets[0]!.close();
    });
    await waitFor(() =>
      expect(screen.getByTestId("connection-status").textContent).toBe("disconnected"),
    );
    await waitFor(() => expect(sockets).toHaveLength(2), { timeout: 1_500 });

    await act(async () => {
      sockets[1]!.open();
    });

    expect(connectEnvelope(sockets[1]!)).toEqual({
      type: "connect",
      data: { sessionId: "session-1" },
    });

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(3));

    await act(async () => {
      sockets[2]!.open();
    });

    expect(connectEnvelope(sockets[2]!)).toEqual({
      type: "connect",
      data: { sessionId: "session-1" },
    });
  });

  it("clears social data on a manual disconnect", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());
    await act(async () => {
      sockets[0]!.message({
        type: "social_state",
        data: {
          userId: "user-1",
          friends: [{ id: "friend-1", name: "Avery", online: true }],
          incomingFriendRequests: [
            {
              id: "request-1",
              user: { id: "friend-2", name: "Blake", online: true },
              createdAt: new Date().toISOString(),
            },
          ],
          outgoingFriendRequestUserIds: [],
          gameInvites: [
            {
              id: "invite-1",
              user: { id: "friend-3", name: "Casey", online: true },
              roomCode: "ROOM42",
              createdAt: new Date().toISOString(),
              expiresAt: new Date(Date.now() + 60_000).toISOString(),
            },
          ],
        },
      });
    });

    expect(screen.getByTestId("social-user-id").textContent).toBe("user-1");
    expect(screen.getByTestId("social-notification-count").textContent).toBe("2");

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));
    });

    expect(screen.getByTestId("connection-status").textContent).toBe("disconnected");
    expect(screen.getByTestId("social-user-id").textContent).toBe("");
    expect(screen.getByTestId("social-notification-count").textContent).toBe("0");
  });

  it("removes an invitation notification when it expires", async () => {
    render(
      <GameWebSocketProvider>
        <Harness>
          <NotificationsDropdown />
        </Harness>
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    const now = Date.now();
    vi.useFakeTimers();
    vi.setSystemTime(now);
    try {
      await act(async () => {
        sockets[0]!.message({
          type: "social_state",
          data: {
            userId: "user-1",
            friends: [],
            incomingFriendRequests: [],
            outgoingFriendRequestUserIds: [],
            gameInvites: [
              {
                id: "invite-1",
                user: { id: "friend-1", name: "Avery", online: true },
                roomCode: "ROOM42",
                createdAt: new Date(now).toISOString(),
                expiresAt: new Date(now + 1_000).toISOString(),
              },
            ],
          },
        });
      });

      expect(screen.getByRole("button", { name: "Notifications" }).textContent).toContain("1");

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1_001);
      });

      expect(screen.getByRole("button", { name: "Notifications" }).textContent).not.toContain("1");
    } finally {
      vi.useRealTimers();
    }
  });

  it("keeps quick-game results when the final game snapshot broadcast is missed", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    const players = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: false,
      },
    ];
    const game = {
      gameMode: "quick",
      phase: 1,
      round: 1,
      dealerIndex: 0,
      roundWinnerIndex: 0,
      turn: { number: 1, playerIndex: 0, playerId: "player-1", hasDrawn: true },
      players: [
        {
          playerId: "player-1",
          handCount: 0,
          totalPoints: 0,
          pointsGained: 0,
          hasOpened: true,
        },
      ],
      hand: [],
      drawPileCount: 20,
      discardPile: [],
      activeCompositions: [],
    };

    await act(async () => {
      sockets[0]!.message({
        type: "game_state",
        data: {
          room: {
            code: "QUICK1",
            phase: "in_progress",
            gameMode: "quick",
            hostPlayerId: "player-1",
            players,
          },
          game,
        },
      });
      sockets[0]!.message({
        type: "room_state",
        data: {
          room: {
            code: "QUICK1",
            phase: "game_over",
            gameMode: "quick",
            hostPlayerId: "player-1",
            players,
          },
        },
      });
      sockets[0]!.message({
        type: "room_state",
        data: {
          room: {
            code: "QUICK1",
            phase: "lobby",
            gameMode: "quick",
            hostPlayerId: "player-1",
            players,
          },
        },
      });
    });

    expect(screen.getByTestId("completed-game-phase").textContent).toBe("game_over");
  });

  it("localizes error codes while retaining the backend message for developers", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    await act(async () => {
      sockets[0]!.message({
        type: "error",
        data: { code: "room_full", message: "room is full" },
      });
    });

    expect(screen.getByTestId("last-error").textContent).toBe("The room is full.");
    expect(screen.getByTestId("last-error-code").textContent).toBe("room_full");
    expect(screen.getByTestId("last-error-message").textContent).toBe("room is full");
  });

  it("rejects inherited object property names as message types", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    await act(async () => {
      sockets[0]!.message({ type: "toString", data: null });
    });

    expect(screen.getByTestId("last-error-code").textContent).toBe("unknown_message_type");
  });

  it("sends the selected card value with discard actions", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    fireEvent.click(screen.getByRole("button", { name: "Discard test card" }));
    fireEvent.click(screen.getByRole("button", { name: "Play and discard test card" }));

    expect(JSON.parse(sockets[0]!.sent.at(-2) ?? "{}")).toEqual({
      type: "discard",
      data: { cardIndex: 4, card: { rank: 12, suit: 2 } },
    });
    expect(JSON.parse(sockets[0]!.sent.at(-1) ?? "{}")).toEqual({
      type: "play_and_discard",
      data: {
        compositions: [],
        additions: [],
        reclaims: [],
        cardIndex: 1,
        card: { isJoker: true },
      },
    });
  });

  it("keeps passive draft rejections silent even after an intervening state broadcast", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    fireEvent.click(screen.getByRole("button", { name: "Sync drafts" }));

    await act(async () => {
      sockets[0]!.message({ type: "game_state", data: { game: null } });
      sockets[0]!.message({
        type: "error",
        data: {
          action: "draft_update",
          code: "not_your_turn",
          message: "not your turn",
        },
      });
    });

    expect(screen.getByTestId("last-error").textContent).toBe("");
    expect(screen.getByTestId("last-error-code").textContent).toBe("");
  });

  it("still presents not-your-turn errors from explicit player actions", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    });
    await waitFor(() => expect(sockets).toHaveLength(1));
    await act(async () => sockets[0]!.open());

    await act(async () => {
      sockets[0]!.message({
        type: "error",
        data: { action: "discard", code: "not_your_turn", message: "not your turn" },
      });
    });

    expect(screen.getByTestId("last-error-code").textContent).toBe("not_your_turn");
  });
});

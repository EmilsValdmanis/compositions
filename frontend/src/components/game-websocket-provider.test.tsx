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

const { GameWebSocketProvider, useGameWebSocket } =
  await import("#/components/game-websocket-provider");

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
  const { state, connect, disconnect } = useGameWebSocket();

  return (
    <div>
      <p data-testid="session-id">{state.sessionId}</p>
      <p data-testid="connection-status">{state.connectionStatus}</p>
      <button type="button" onClick={() => void connect()}>
        Connect
      </button>
      <button type="button" onClick={disconnect}>
        Disconnect
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
});

// @vitest-environment jsdom
import { createRequire } from "node:module";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { GameWebSocketProvider, useGameWebSocket } from "#/components/game-websocket-provider";

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

vi.mock("#/lib/game-auth", () => ({
  getGameConnectionAuth: vi.fn().mockResolvedValue({ playerId: "player-1" }),
}));

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
    sockets.splice(0);
    vi.stubEnv("VITE_GAME_SERVER_URL", "ws://localhost:8080");
    Object.defineProperty(globalThis, "WebSocket", {
      value: FakeWebSocket,
      configurable: true,
    });
  });

  it("restores the server session id when reconnecting", async () => {
    render(
      <GameWebSocketProvider>
        <Harness />
      </GameWebSocketProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    await waitFor(() => expect(sockets).toHaveLength(1));

    sockets[0]!.open();
    expect(connectEnvelope(sockets[0]!)).toEqual({
      type: "connect",
      data: { sessionId: "" },
    });

    sockets[0]!.message({
      type: "connected",
      data: { sessionId: "session-1", playerId: "player-1" },
    });
    await waitFor(() => expect(screen.getByTestId("session-id").textContent).toBe("session-1"));

    fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    await waitFor(() => expect(sockets).toHaveLength(2));

    sockets[1]!.open();

    expect(connectEnvelope(sockets[1]!)).toEqual({
      type: "connect",
      data: { sessionId: "session-1" },
    });
  });
});

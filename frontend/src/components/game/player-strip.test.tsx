// @vitest-environment jsdom
import { createRequire } from "node:module";
import type { PlayerSnapshot } from "#/components/game-websocket-provider";

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
}

const { cleanup, render, screen } = await import("@testing-library/react");
const { afterEach, describe, expect, it } = await import("vite-plus/test");
const { PlayerStrip } = await import("#/components/game/player-strip");

afterEach(cleanup);

describe("PlayerStrip", () => {
  it("shows a skull instead of the turn spinner for a forfeited player", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: false,
        seat: 0,
        isHost: false,
        canReconnect: false,
        forfeited: true,
      },
    ];

    render(<PlayerStrip players={players} game={null} />);

    expect(screen.getByLabelText("Avery forfeited")).toBeTruthy();
    expect(screen.queryByLabelText("Avery's turn")).toBeNull();
  });
});

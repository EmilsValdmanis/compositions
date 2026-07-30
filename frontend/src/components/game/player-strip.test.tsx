// @vitest-environment jsdom
import { createRequire } from "node:module";
import type { GameSnapshot, PlayerSnapshot } from "#/components/game-websocket-provider";

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
  it("composes player rows from item primitives", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: false,
        canReconnect: false,
      },
    ];

    const { container } = render(<PlayerStrip players={players} game={null} />);

    expect(container.querySelector('[data-slot="item-group"]')).toBeTruthy();
    expect(container.querySelector('[data-slot="item"]')).toBeTruthy();
    expect(container.querySelector('[data-slot="item-media"]')).toBeTruthy();
    expect(container.querySelector('[data-slot="item-content"]')).toBeTruthy();
    expect(container.querySelector('[data-slot="item-actions"]')).toBeTruthy();
  });

  it("uses a subtle shaded surface and glow for the active turn", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: false,
        canReconnect: false,
      },
    ];
    const game = {
      turn: { playerId: "player-1" },
      players: [{ playerId: "player-1", handCount: 9 }],
    } as GameSnapshot;

    const { container } = render(<PlayerStrip players={players} game={game} />);

    const activePlayer = container.querySelector('[data-slot="item"]');

    expect(activePlayer?.getAttribute("data-variant")).toBe("outline");
    expect(activePlayer?.getAttribute("data-active-turn")).toBe("true");
    expect(activePlayer?.getAttribute("aria-current")).toBe("true");
    expect(activePlayer?.className).toContain("player-turn-surface-active");
    expect(container.querySelector('[data-slot="avatar-badge"]')?.className).toContain("ring-card");
    expect(screen.getByLabelText("Avery's turn")).toBeTruthy();
    expect(container.querySelector('[data-slot="turn-token"]')).toBeNull();
  });

  it("uses compact, borderless rows in a menu presentation", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: false,
        canReconnect: false,
      },
    ];
    const game = {
      turn: { playerId: "player-1" },
      players: [{ playerId: "player-1", handCount: 9 }],
    } as GameSnapshot;

    const { container } = render(<PlayerStrip players={players} game={game} presentation="menu" />);

    expect(
      container.querySelector('[data-slot="item-group"]')?.getAttribute("data-presentation"),
    ).toBe("menu");
    expect(container.querySelector('[data-slot="item"]')?.getAttribute("data-variant")).toBe(
      "default",
    );
    expect(container.querySelector('[data-slot="avatar"]')?.className).toContain("size-8");
  });

  it("shows a skull instead of the active-turn treatment for a forfeited player", () => {
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

  it("anchors card transfers to the player's card-count badge", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: false,
        canReconnect: false,
      },
    ];
    const game = {
      turn: { playerId: "player-2" },
      players: [{ playerId: "player-1", handCount: 9 }],
    } as GameSnapshot;

    render(<PlayerStrip players={players} game={game} />);

    expect(document.querySelector('[data-card-motion-player="player-1"]')).toBe(
      screen.getByLabelText("9 cards"),
    );
  });

  it("renders active emotes outside the scrollable player list", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: false,
        canReconnect: false,
        activeEmote: {
          id: "emote-1",
          emoji: "🔥",
          expiresAt: new Date(Date.now() + 4000).toISOString(),
        },
      },
    ];

    render(<PlayerStrip players={players} game={null} />);

    const emote = screen.getByLabelText("Player emote");
    expect(emote.parentElement).toBe(document.body);
    expect(emote.classList.contains("fixed")).toBe(true);
  });
});

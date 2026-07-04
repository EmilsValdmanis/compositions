// @vitest-environment jsdom
import { createRequire } from "node:module";
import { act, render, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vite-plus/test";
import { GameBoardView } from "#/components/game/game-board-view";
import {
  type ActionResult,
  type GameSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";

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

let dndContextProps: {
  onDragStart?: (event: any) => void;
  onDragEnd?: (event: any) => void;
} = {};

vi.mock("#/components/game-websocket-provider", () => ({
  useGameWebSocket: () => ({
    updateTurnDrafts: vi.fn(),
  }),
}));

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children, ...props }: { children: ReactNode }) => {
    dndContextProps = props;
    return <>{children}</>;
  },
  DragOverlay: ({ children }: { children: ReactNode }) => <>{children}</>,
  closestCenter: () => [],
  pointerWithin: () => [],
  useDndContext: () => ({ active: null, over: null }),
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
  useDroppable: () => ({
    setNodeRef: vi.fn(),
    isOver: false,
  }),
}));

vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: ({ children }: { children: ReactNode }) => <>{children}</>,
  horizontalListSortingStrategy: vi.fn(),
  arrayMove: <T,>(items: T[], from: number, to: number) => {
    const next = [...items];
    const [item] = next.splice(from, 1);
    if (item !== undefined) {
      next.splice(to, 0, item);
    }
    return next;
  },
  useSortable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    transition: undefined,
    isDragging: false,
  }),
}));

function makeGame(hand: GameSnapshot["hand"]): GameSnapshot {
  return {
    phase: 1,
    round: 1,
    dealerIndex: 0,
    roundWinnerIndex: -1,
    turn: {
      number: 1,
      playerIndex: 0,
      playerId: "player-1",
      hasDrawn: true,
      mustUseDiscardDraw: false,
    },
    players: [
      {
        playerId: "player-1",
        handCount: hand.length,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: true,
      },
    ],
    hand,
    drawPileCount: 20,
    discardPile: [{ rank: 2, suit: 0 }],
    activeCompositions: [],
  };
}

const players: PlayerSnapshot[] = [
  {
    playerId: "player-1",
    name: "Avery",
    connected: true,
    seat: 0,
    isHost: true,
    canReconnect: false,
  },
];

function dragStart(handKey: string, cardIndex: number) {
  dndContextProps.onDragStart?.({
    active: {
      id: handKey,
      data: {
        current: {
          cardIndex,
        },
      },
    },
  });
}

function dragEnd(handKey: string, overId: string, cardIndex: number) {
  dndContextProps.onDragEnd?.({
    active: {
      id: handKey,
      data: {
        current: {
          cardIndex,
        },
      },
    },
    over: {
      id: overId,
    },
  });
}

describe("GameBoardView discard drops", () => {
  it("does not discard with a post-submit hand index when the staged table play fails", async () => {
    const onDiscardCard = vi.fn<() => Promise<ActionResult>>().mockResolvedValue({
      action: "discard_card",
      playerId: "player-1",
      ok: true,
    });
    const onPlayTable = vi.fn<() => Promise<ActionResult>>().mockResolvedValue({
      action: "play_table",
      playerId: "player-1",
      ok: false,
    });

    render(
      <GameBoardView
        game={makeGame([
          { rank: 1, suit: 0 },
          { rank: 13, suit: 3 },
          { rank: 12, suit: 2 },
          { rank: 11, suit: 1 },
        ])}
        roomCode="ROOM"
        playerId="player-1"
        players={players}
        connectedPlayers={1}
        turnState={{
          canDrawDeck: false,
          canDrawDiscard: false,
          canDiscard: true,
          isMyTurn: true,
          turnPlayerName: "Avery",
        }}
        topDiscardCard={{ rank: 2, suit: 0 }}
        onDiscardCard={onDiscardCard}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={onPlayTable}
        disableDraftSync
      />,
    );

    await act(async () => {
      dragStart("1-0-1", 0);
    });
    await act(async () => {
      dragEnd("1-0-1", "new-composition-drop-zone", 0);
    });
    await act(async () => {
      dragStart("12-2-1", 2);
    });
    await act(async () => {
      dragEnd("12-2-1", "discard-pile", 2);
    });

    await waitFor(() => expect(onPlayTable).toHaveBeenCalledTimes(1));
    expect(onDiscardCard).not.toHaveBeenCalled();
  });
});

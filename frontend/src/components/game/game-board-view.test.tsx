// @vitest-environment jsdom
import { createRequire } from "node:module";
import { type ComponentProps, type ReactNode } from "react";
import {
  type ActionResult,
  type CardSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type TablePlayRequest,
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
  globalThis.Element = dom.window.Element;
  globalThis.HTMLElement = dom.window.HTMLElement;
  globalThis.Node = dom.window.Node;
  globalThis.MutationObserver = dom.window.MutationObserver;
  globalThis.getComputedStyle = dom.window.getComputedStyle.bind(dom.window);
  Object.defineProperty(globalThis, "navigator", {
    value: dom.window.navigator,
    configurable: true,
  });
}

const { act, cleanup, fireEvent, render, waitFor } = await import("@testing-library/react");
const { afterEach, beforeEach, describe, expect, it, vi } = await import("vite-plus/test");

let dndContextProps: {
  onDragStart?: (event: any) => void;
  onDragOver?: (event: any) => void;
  onDragEnd?: (event: any) => void;
} = {};
let sortableContextItems: string[][] = [];

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
  SortableContext: ({ children, items }: { children: ReactNode; items?: string[] }) => {
    if (items) {
      sortableContextItems.push(items);
    }
    return <>{children}</>;
  },
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

const { GameBoardView } = await import("#/components/game/game-board-view");

beforeEach(() => {
  cleanup();
  dndContextProps = {};
  sortableContextItems = [];
});

afterEach(() => {
  cleanup();
  dndContextProps = {};
  sortableContextItems = [];
});

function makeGame(
  hand: GameSnapshot["hand"],
  overrides: Partial<GameSnapshot> & {
    players?: GameSnapshot["players"];
    activeCompositions?: GameSnapshot["activeCompositions"];
  } = {},
): GameSnapshot {
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
    ...overrides,
  };
}

const players: PlayerSnapshot[] = [
  {
    playerId: "player-1",
    name: "Avery",
    imageUrl: "https://cdn.example.com/avery.png",
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

function dragOver(handKey: string, overId: string) {
  dndContextProps.onDragOver?.({
    active: {
      id: handKey,
    },
    over: {
      id: overId,
    },
  });
}

describe("GameBoardView draft returns", () => {
  it("opens a sortable hand gap while returning a draft card and restores order on reset", async () => {
    const view = render(
      <GameBoardView
        game={makeGame([
          { rank: 1, suit: 0 },
          { rank: 2, suit: 0 },
          { rank: 3, suit: 0 },
        ])}
        roomCode="RESET-ROOM"
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
        topDiscardCard={{ rank: 4, suit: 0 }}
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        disableDraftSync
      />,
    );

    await act(async () => {
      dragStart("2-0-1", 1);
    });
    await act(async () => {
      dragEnd("2-0-1", "new-composition-drop-zone", 1);
    });
    await act(async () => {
      dragStart("3-0-1", 2);
    });
    await act(async () => {
      dragEnd("3-0-1", "2-0-1", 2);
    });

    sortableContextItems = [];
    await act(async () => {
      dragStart("2-0-1", 1);
    });
    await act(async () => {
      dragOver("2-0-1", "1-0-1");
    });

    expect(sortableContextItems.at(-1)).toEqual(["2-0-1", "1-0-1"]);

    await act(async () => {
      dragEnd("2-0-1", "1-0-1", 1);
    });
    expect(sortableContextItems.at(-1)).toEqual(["2-0-1", "1-0-1"]);

    fireEvent.click(view.getByRole("button", { name: "Reset" }));

    expect(sortableContextItems.at(-1)).toEqual(["1-0-1", "2-0-1", "3-0-1"]);
  });
});

describe("GameBoardView discard drops", () => {
  it("shows the turn-start cue only for the active player", () => {
    const sharedProps = {
      game: makeGame([]),
      roomCode: "ROOM",
      playerId: "player-1",
      players,
      connectedPlayers: 1,
      topDiscardCard: { rank: 2, suit: 0 },
      onDiscardCard: vi.fn(),
      onDrawFromDeck: vi.fn(),
      onDrawFromDiscard: vi.fn(),
      onPlayTable: vi.fn(),
      onPlayTableAndDiscard: vi.fn(),
      onSendEmote: vi.fn(),
      disableDraftSync: true,
    } satisfies Omit<ComponentProps<typeof GameBoardView>, "turnState">;
    const view = render(
      <GameBoardView
        {...sharedProps}
        turnState={{
          canDrawDeck: false,
          canDrawDiscard: false,
          canDiscard: true,
          isMyTurn: false,
          turnPlayerName: "Avery",
        }}
      />,
    );

    expect(view.queryByText("Your turn")).toBeNull();

    view.rerender(
      <GameBoardView
        {...sharedProps}
        turnState={{
          canDrawDeck: false,
          canDrawDiscard: false,
          canDiscard: true,
          isMyTurn: true,
          turnPlayerName: "Avery",
        }}
      />,
    );

    const cue = view.getByRole("status", { name: "Your turn, Avery" });
    expect(cue?.getAttribute("data-turn-number")).toBe("1");
    expect(cue.querySelector('[data-slot="avatar"]')).not.toBeNull();
    expect(view.queryByText("Play or discard")).toBeNull();
    expect(cue.textContent).toContain("R01");
    expect(cue.textContent).toContain("T01");
  });

  it("commits a staged table play and discard as one action", async () => {
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
    const onPlayTableAndDiscard = vi
      .fn<
        (play: TablePlayRequest, cardIndex: number, card: CardSnapshot) => Promise<ActionResult>
      >()
      .mockResolvedValue({
        action: "play_and_discard",
        playerId: "player-1",
        ok: true,
      });

    const view = render(
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
        onPlayTableAndDiscard={onPlayTableAndDiscard}
        onSendEmote={vi.fn()}
        disableDraftSync
      />,
    );

    expect(view.getByLabelText("Avery's turn")).toBeTruthy();

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

    await waitFor(() => expect(onPlayTableAndDiscard).toHaveBeenCalledTimes(1));
    expect(onPlayTableAndDiscard.mock.calls[0]?.[1]).toBe(1);
    expect(onPlayTableAndDiscard.mock.calls[0]?.[2]).toEqual({ rank: 12, suit: 2 });
    expect(onPlayTable).not.toHaveBeenCalled();
    expect(onDiscardCard).not.toHaveBeenCalled();
  });

  it("does not discard an unopened player's reclaim-only draft", async () => {
    const onDiscardCard = vi.fn<() => Promise<ActionResult>>().mockResolvedValue({
      action: "discard_card",
      playerId: "player-1",
      ok: true,
    });
    const onPlayTable = vi.fn<() => Promise<ActionResult>>().mockResolvedValue({
      action: "play_table",
      playerId: "player-1",
      ok: true,
    });

    render(
      <GameBoardView
        game={makeGame(
          [
            { rank: 6, suit: 0 },
            { rank: 2, suit: 3 },
          ],
          {
            players: [
              {
                playerId: "player-1",
                handCount: 2,
                totalPoints: 0,
                pointsGained: 0,
                hasOpened: false,
              },
            ],
            activeCompositions: [
              {
                type: "run",
                cards: [{ rank: 5, suit: 0 }, { isJoker: true }, { rank: 7, suit: 0 }],
                jokerRepresentations: {
                  1: [{ rank: 6, suit: 0 }],
                },
                points: 18,
                complete: false,
              },
            ],
          },
        )}
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
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        disableDraftSync
      />,
    );

    await act(async () => {
      dragStart("6-0-1", 0);
    });
    await act(async () => {
      dragEnd("6-0-1", "table-composition-joker-0-1", 0);
    });
    await act(async () => {
      dragStart("2-3-1", 1);
    });
    await act(async () => {
      dragEnd("2-3-1", "discard-pile", 1);
    });

    expect(onPlayTable).not.toHaveBeenCalled();
    expect(onDiscardCard).not.toHaveBeenCalled();
  });
});

describe("GameBoardView spectator turn drafts", () => {
  it("shows a duplicate-key joker reclaim draft as reclaim instead of add", () => {
    const view = render(
      <GameBoardView
        game={{
          ...makeGame([]),
          turn: {
            number: 1,
            playerIndex: 1,
            playerId: "player-2",
            hasDrawn: true,
            mustUseDiscardDraw: false,
          },
          activeCompositions: [
            {
              type: "run",
              cards: [
                { isJoker: true },
                { rank: 4, suit: 1 },
                { rank: 5, suit: 1 },
                { rank: 6, suit: 1 },
                { rank: 7, suit: 1 },
              ],
              jokerRepresentations: {
                0: [{ rank: 3, suit: 1 }],
              },
              points: 30,
              complete: false,
            },
          ],
          turnActivity: {
            playerId: "player-2",
            round: 1,
            turnNumber: 1,
            baselineCompositions: [
              {
                type: "run",
                cards: [
                  { isJoker: true },
                  { rank: 4, suit: 1 },
                  { rank: 5, suit: 1 },
                  { rank: 6, suit: 1 },
                  { rank: 7, suit: 1 },
                ],
                jokerRepresentations: {
                  0: [{ rank: 3, suit: 1 }],
                },
                points: 30,
                complete: false,
              },
            ],
            draftCompositions: [
              {
                tableIndex: 0,
                cards: [{ rank: 3, suit: 1 }],
                reclaimTargets: {
                  "3-1-2": 0,
                },
              },
            ],
          },
        }}
        roomCode="ROOM"
        playerId="player-1"
        players={[
          ...players,
          {
            playerId: "player-2",
            name: "Blair",
            connected: true,
            seat: 1,
            isHost: false,
            canReconnect: false,
          },
        ]}
        connectedPlayers={2}
        turnState={{
          canDrawDeck: false,
          canDrawDiscard: false,
          canDiscard: false,
          isMyTurn: false,
          turnPlayerName: "Blair",
        }}
        topDiscardCard={{ rank: 2, suit: 0 }}
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        disableDraftSync
      />,
    );

    expect(view.getByText("Reclaim")).toBeTruthy();
    expect(view.queryByText("Add")).toBeNull();
  });
});

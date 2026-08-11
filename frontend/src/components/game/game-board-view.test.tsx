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
  sensors?: Array<{ options?: Record<string, unknown> }>;
  onDragStart?: (event: any) => void;
  onDragOver?: (event: any) => void;
  onDragEnd?: (event: any) => void;
  onDragCancel?: (event: any) => void;
} = {};
let dragOverlayProps: {
  modifiers?: Array<(args: Record<string, unknown>) => Record<string, unknown>>;
} = {};
let sortableContextItems: string[][] = [];

vi.mock("#/components/game-websocket-provider", () => ({
  useGameWebSocket: () => ({
    updateTurnDrafts: vi.fn(),
  }),
}));

vi.mock("@dnd-kit/core", () => ({
  KeyboardSensor: class KeyboardSensor {},
  MouseSensor: class MouseSensor {},
  TouchSensor: class TouchSensor {},
  DndContext: ({ children, ...props }: { children: ReactNode }) => {
    dndContextProps = props;
    return <>{children}</>;
  },
  DragOverlay: ({
    children,
    ...props
  }: {
    children: ReactNode;
    modifiers?: Array<(args: Record<string, unknown>) => Record<string, unknown>>;
  }) => {
    dragOverlayProps = props;
    return <>{children}</>;
  },
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
  useSensor: (_sensor: unknown, options?: Record<string, unknown>) => ({ options }),
  useSensors: (...sensors: Array<{ options?: Record<string, unknown> }>) => sensors,
}));

vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: ({ children, items }: { children: ReactNode; items?: string[] }) => {
    if (items) {
      sortableContextItems.push(items);
    }
    return <>{children}</>;
  },
  horizontalListSortingStrategy: vi.fn(),
  sortableKeyboardCoordinates: vi.fn(),
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

const toastInfoMock = vi.fn();
vi.doMock("sonner", () => ({
  toast: {
    error: vi.fn(),
    info: toastInfoMock,
    success: vi.fn(),
  },
}));

const { GameBoardView } = await import("#/components/game/game-board-view");

beforeEach(() => {
  cleanup();
  dndContextProps = {};
  dragOverlayProps = {};
  sortableContextItems = [];
  toastInfoMock.mockClear();
});

afterEach(() => {
  cleanup();
  dndContextProps = {};
  dragOverlayProps = {};
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

describe("GameBoardView drag sensors", () => {
  it("keeps touch scrolling available until a deliberate hold starts dragging", () => {
    const view = render(
      <GameBoardView
        game={makeGame([{ rank: 1, suit: 0 }])}
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
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        draftSyncMode="disabled"
      />,
    );

    expect(dndContextProps.sensors?.[0]?.options).toEqual({
      activationConstraint: { distance: 4 },
    });
    expect(dndContextProps.sensors?.[1]?.options).toEqual({
      activationConstraint: { delay: 180, tolerance: 8 },
    });
    expect(dndContextProps.sensors?.[2]?.options).toHaveProperty("coordinateGetter");
    expect(
      dragOverlayProps.modifiers?.[0]?.({
        overlayNodeRect: { left: -4, right: 52, top: -6, bottom: 74 },
        transform: { x: 0, y: 0, scaleX: 1, scaleY: 1 },
        windowRect: { left: 0, right: 800, top: 0, bottom: 600 },
      }),
    ).toMatchObject({ x: 12, y: 14 });
    expect(view.container.querySelector('[data-slot="turn-start-cue"]')).toBeNull();
    expect(toastInfoMock).toHaveBeenCalledWith("Your turn", {
      id: "your-turn:player-1:1:1",
      duration: 1_600,
    });
    const handCards = view.container.querySelector('[data-onboarding-target="hand-cards"]');
    expect(handCards?.className).toContain("w-max");
    expect(handCards?.parentElement?.className).toContain("justify-center");
  });
});

describe("GameBoardView guided interaction callbacks", () => {
  it("reports when a deck draw has settled into the hand", async () => {
    const onDrawFromDeck = vi.fn();
    const onDrawDragStateChange = vi.fn();
    const onDrawSettled = vi.fn();
    const initialGame = makeGame([{ rank: 5, suit: 3 }], {
      turn: {
        number: 1,
        playerIndex: 0,
        playerId: "player-1",
        hasDrawn: false,
        mustUseDiscardDraw: false,
      },
    });
    const boardProps: ComponentProps<typeof GameBoardView> = {
      game: initialGame,
      roomCode: "GUIDED-DRAW",
      playerId: "player-1",
      players,
      connectedPlayers: 1,
      turnState: {
        canDrawDeck: true,
        canDrawDiscard: false,
        canDiscard: false,
        isMyTurn: true,
        turnPlayerName: "Avery",
      },
      topDiscardCard: { rank: 2, suit: 0 },
      onDiscardCard: vi.fn(),
      onDrawFromDeck,
      onDrawFromDiscard: vi.fn(),
      onPlayTable: vi.fn(),
      onPlayTableAndDiscard: vi.fn(),
      onSendEmote: vi.fn(),
      draftSyncMode: "disabled",
      guidance: {
        stage: "draw",
        onDrawDragStateChange,
        onDrawSettled,
        onDraftStateChange: vi.fn(),
      },
    };

    const view = render(<GameBoardView {...boardProps} />);
    expect(onDrawDragStateChange).not.toHaveBeenCalled();

    await act(async () => {
      dndContextProps.onDragStart?.({
        active: { id: "draw-pile", data: { current: { drawSource: "deck" } } },
      });
    });
    expect(onDrawDragStateChange).toHaveBeenLastCalledWith(true);

    view.rerender(
      <GameBoardView
        {...boardProps}
        game={makeGame(
          [
            { rank: 5, suit: 3 },
            { rank: 7, suit: 3 },
          ],
          { turn: { ...initialGame.turn, hasDrawn: true } },
        )}
      />,
    );

    await act(async () => {
      dndContextProps.onDragEnd?.({
        active: { id: "draw-pile", data: { current: { drawSource: "deck" } } },
        over: { id: "5-3-1" },
      });
    });

    expect(onDrawFromDeck).toHaveBeenCalledTimes(1);
    expect(onDrawSettled).toHaveBeenCalledTimes(1);
    expect(onDrawDragStateChange).toHaveBeenLastCalledWith(false);
  });

  it("does not strand a committed draw when the drag is canceled", async () => {
    const onDrawSettled = vi.fn();
    const onDrawDragStateChange = vi.fn();
    const initialGame = makeGame([{ rank: 5, suit: 3 }], {
      turn: {
        number: 1,
        playerIndex: 0,
        playerId: "player-1",
        hasDrawn: false,
        mustUseDiscardDraw: false,
      },
    });
    const boardProps: ComponentProps<typeof GameBoardView> = {
      game: initialGame,
      roomCode: "GUIDED-DRAW-CANCEL",
      playerId: "player-1",
      players,
      connectedPlayers: 1,
      turnState: {
        canDrawDeck: true,
        canDrawDiscard: false,
        canDiscard: false,
        isMyTurn: true,
        turnPlayerName: "Avery",
      },
      topDiscardCard: { rank: 2, suit: 0 },
      onDiscardCard: vi.fn(),
      onDrawFromDeck: vi.fn(),
      onDrawFromDiscard: vi.fn(),
      onPlayTable: vi.fn(),
      onPlayTableAndDiscard: vi.fn(),
      onSendEmote: vi.fn(),
      draftSyncMode: "disabled",
      guidance: {
        stage: "draw",
        onDrawDragStateChange,
        onDrawSettled,
        onDraftStateChange: vi.fn(),
      },
    };

    const view = render(<GameBoardView {...boardProps} />);
    await act(async () => {
      dndContextProps.onDragStart?.({
        active: { id: "draw-pile", data: { current: { drawSource: "deck" } } },
      });
    });
    view.rerender(
      <GameBoardView
        {...boardProps}
        game={makeGame(
          [
            { rank: 5, suit: 3 },
            { rank: 7, suit: 3 },
          ],
          { turn: { ...initialGame.turn, hasDrawn: true } },
        )}
      />,
    );

    await act(async () => dndContextProps.onDragCancel?.({}));

    expect(onDrawSettled).toHaveBeenCalledOnce();
    expect(onDrawDragStateChange).toHaveBeenLastCalledWith(false);
  });

  it("reports when a staged composition becomes valid", async () => {
    const onDraftStateChange = vi.fn();

    render(
      <GameBoardView
        game={makeGame([
          { rank: 5, suit: 3 },
          { rank: 6, suit: 3 },
          { rank: 7, suit: 3 },
          { rank: 12, suit: 0 },
        ])}
        roomCode="GUIDED-COMPOSITION"
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
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        draftSyncMode="disabled"
        guidance={{
          stage: "compose",
          onDrawDragStateChange: vi.fn(),
          onDrawSettled: vi.fn(),
          onDraftStateChange,
        }}
      />,
    );

    await act(async () => dragStart("5-3-1", 0));
    await act(async () => dragEnd("5-3-1", "new-composition-drop-zone", 0));
    await act(async () => dragStart("6-3-1", 1));
    await act(async () => dragEnd("6-3-1", "5-3-1", 1));
    await act(async () => dragStart("7-3-1", 2));
    await act(async () => dragEnd("7-3-1", "6-3-1", 2));

    await waitFor(() =>
      expect(onDraftStateChange).toHaveBeenCalledWith(
        expect.objectContaining({
          hasDraftedCompositions: true,
          canSubmitTablePlay: true,
          isDraggingCard: false,
        }),
      ),
    );
    const finalDraftState = onDraftStateChange.mock.calls.at(-1)?.[0];
    expect(finalDraftState?.draftCompositions).toHaveLength(1);
    expect(finalDraftState?.draftCompositions[0]?.cards).toEqual(
      expect.arrayContaining([
        { rank: 5, suit: 3 },
        { rank: 6, suit: 3 },
        { rank: 7, suit: 3 },
      ]),
    );
  });
});

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
        draftSyncMode="disabled"
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

describe("GameBoardView hand ordering", () => {
  it("reorders a joker while it is dragged across the hand", async () => {
    render(
      <GameBoardView
        game={makeGame([{ rank: 4, suit: 0 }, { isJoker: true }, { rank: 6, suit: 0 }])}
        roomCode="JOKER-ORDER"
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
        topDiscardCard={{ rank: 7, suit: 0 }}
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        draftSyncMode="disabled"
      />,
    );

    await act(async () => dragStart("joker-1", 1));
    await act(async () => dragOver("joker-1", "6-0-1"));
    await act(async () => dragOver("joker-1", "6-0-1"));

    expect(sortableContextItems.at(-1)).toEqual(["4-0-1", "6-0-1", "joker-1"]);
  });

  it("handles a repeated collision target only once while reordering a draft", async () => {
    const onDraftStateChange = vi.fn();

    render(
      <GameBoardView
        game={makeGame([
          { rank: 5, suit: 3 },
          { rank: 6, suit: 3 },
          { rank: 7, suit: 3 },
        ])}
        roomCode="STABLE-DRAFT-ORDER"
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
        topDiscardCard={{ rank: 8, suit: 3 }}
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={vi.fn()}
        onSendEmote={vi.fn()}
        draftSyncMode="disabled"
        guidance={{
          stage: "compose",
          onDrawDragStateChange: vi.fn(),
          onDrawSettled: vi.fn(),
          onDraftStateChange,
        }}
      />,
    );

    await act(async () => dragStart("5-3-1", 0));
    await act(async () => dragEnd("5-3-1", "new-composition-drop-zone", 0));
    await act(async () => dragStart("6-3-1", 1));
    await act(async () => dragEnd("6-3-1", "5-3-1", 1));
    await act(async () => dragStart("7-3-1", 2));
    await act(async () => dragEnd("7-3-1", "6-3-1", 2));

    onDraftStateChange.mockClear();
    await act(async () => dragStart("5-3-1", 0));
    await act(async () => dragOver("5-3-1", "7-3-1"));
    await act(async () => dragOver("5-3-1", "7-3-1"));

    expect(onDraftStateChange).toHaveBeenCalledTimes(1);
    expect(onDraftStateChange.mock.calls[0]?.[0].draftCompositions[0]?.cards).toEqual([
      { rank: 5, suit: 3 },
      { rank: 7, suit: 3 },
      { rank: 6, suit: 3 },
    ]);
  });
});

describe("GameBoardView discard drops", () => {
  it("shows the turn-start toast only for the active player", () => {
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
      draftSyncMode: "disabled",
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

    expect(toastInfoMock).not.toHaveBeenCalled();

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

    expect(toastInfoMock).toHaveBeenCalledTimes(1);
    expect(toastInfoMock).toHaveBeenCalledWith("Your turn", {
      id: "your-turn:player-1:1:1",
      duration: 1_600,
    });
    expect(view.container.querySelector('[data-slot="turn-start-cue"]')).toBeNull();

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

    expect(toastInfoMock).toHaveBeenCalledTimes(1);
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
        draftSyncMode="disabled"
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

  it("marks an invalid composition after the combined discard is rejected", async () => {
    const onPlayTableAndDiscard = vi
      .fn<
        (play: TablePlayRequest, cardIndex: number, card: CardSnapshot) => Promise<ActionResult>
      >()
      .mockRejectedValue(new Error("not a valid composition"));
    const view = render(
      <GameBoardView
        game={makeGame([
          { rank: 5, suit: 0 },
          { rank: 7, suit: 0 },
          { rank: 8, suit: 0 },
          { rank: 2, suit: 1 },
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
        topDiscardCard={{ rank: 3, suit: 0 }}
        onDiscardCard={vi.fn()}
        onDrawFromDeck={vi.fn()}
        onDrawFromDiscard={vi.fn()}
        onPlayTable={vi.fn()}
        onPlayTableAndDiscard={onPlayTableAndDiscard}
        onSendEmote={vi.fn()}
        draftSyncMode="disabled"
      />,
    );

    for (const [handKey, cardIndex, target] of [
      ["5-0-1", 0, "new-composition-drop-zone"],
      ["7-0-1", 1, "5-0-1"],
      ["8-0-1", 2, "5-0-1"],
    ] as const) {
      await act(async () => dragStart(handKey, cardIndex));
      await act(async () => dragEnd(handKey, target, cardIndex));
    }

    await act(async () => dragStart("2-1-1", 3));
    await act(async () => dragEnd("2-1-1", "discard-pile", 3));

    await waitFor(() => expect(onPlayTableAndDiscard).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(view.container.querySelector('[aria-invalid="true"]')).not.toBeNull(),
    );
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
        draftSyncMode="disabled"
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
                id: "draft-table-0",
                tableIndex: 0,
                cards: [{ rank: 3, suit: 1 }],
                cardPlacements: [{ reclaimJokerIndex: 0 }],
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
        draftSyncMode="disabled"
      />,
    );

    expect(view.getByText("Reclaim")).toBeTruthy();
    expect(view.queryByText("Add")).toBeNull();
  });
});

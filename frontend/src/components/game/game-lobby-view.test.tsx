// @vitest-environment jsdom
import { createRequire } from "node:module";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vite-plus/test";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import {
  type CompletedGameSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
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
  onDragEnd?: (event: any) => void;
} = {};

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children, ...props }: { children: ReactNode }) => {
    dndContextProps = props;
    return <>{children}</>;
  },
  closestCenter: vi.fn(),
}));

vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: ({ children }: { children: ReactNode }) => <>{children}</>,
  verticalListSortingStrategy: vi.fn(),
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

const players: PlayerSnapshot[] = [
  {
    playerId: "player-1",
    name: "Avery",
    connected: true,
    seat: 0,
    isHost: true,
    canReconnect: false,
  },
  {
    playerId: "player-2",
    name: "Blake",
    connected: true,
    seat: 1,
    isHost: false,
    canReconnect: false,
  },
  {
    playerId: "player-3",
    name: "Casey",
    connected: true,
    seat: 2,
    isHost: false,
    canReconnect: false,
  },
];

function makeRoom(overrides: Partial<RoomSnapshot> = {}): RoomSnapshot {
  return {
    code: "ABCD12",
    phase: "lobby",
    hostPlayerId: "player-1",
    players,
    ...overrides,
  };
}

function renderLobby(overrides: Partial<React.ComponentProps<typeof GameLobbyView>> = {}) {
  const props: React.ComponentProps<typeof GameLobbyView> = {
    room: null,
    game: null as GameSnapshot | null,
    completedGame: null as CompletedGameSnapshot | null,
    players: [],
    roomCode: "",
    playerId: "player-1",
    roomActions: {
      canCreateRoom: true,
      canJoinRoom: true,
      canLeaveRoom: false,
      canStartGame: false,
    },
    dealChoice: {
      pendingDealChoice: null,
      dealChooserName: null,
      isDealChooser: false,
    },
    onRoomCodeChange: vi.fn(),
    onCreateRoom: vi.fn(),
    onJoinRoom: vi.fn(),
    onStartGame: vi.fn(),
    onChooseDealing: vi.fn(),
    onLeaveRoom: vi.fn(),
    onCopyRoomCode: vi.fn(),
    onCopyRoomLink: vi.fn(),
    roomLink: null,
    ...overrides,
  };

  render(<GameLobbyView {...props} />);
  return props;
}

describe("GameLobbyView", () => {
  it("supports the create and join lobby flow", () => {
    const onCreateRoom = vi.fn();
    const onJoinRoom = vi.fn();
    const onRoomCodeChange = vi.fn();

    renderLobby({
      roomCode: "ROOM42",
      onCreateRoom,
      onJoinRoom,
      onRoomCodeChange,
    });

    fireEvent.click(screen.getByRole("button", { name: "Create room" }));
    fireEvent.change(screen.getByPlaceholderText("Room code"), {
      target: { value: "game7" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Join" }));

    expect(onCreateRoom).toHaveBeenCalledTimes(1);
    expect(onRoomCodeChange).toHaveBeenCalledWith("GAME7");
    expect(onJoinRoom).toHaveBeenCalledWith("ROOM42");
  });

  it("renders room state and non-chooser deal status", () => {
    renderLobby({
      room: makeRoom({
        pendingDealChoice: {
          dealerIndex: 0,
          chooserIndex: 2,
          chooserPlayerId: "player-3",
        },
      }),
      players,
      playerId: "player-2",
      roomActions: {
        canCreateRoom: false,
        canJoinRoom: false,
        canLeaveRoom: true,
        canStartGame: false,
      },
      dealChoice: {
        pendingDealChoice: {
          dealerIndex: 0,
          chooserIndex: 2,
          chooserPlayerId: "player-3",
        },
        dealChooserName: "Casey",
        isDealChooser: false,
      },
      roomLink: "http://localhost/?room=ABCD12",
    });

    expect(screen.getByText("ABCD12")).toBeTruthy();
    expect(screen.getByText("Avery")).toBeTruthy();
    expect(screen.getByText(/Waiting for/).textContent).toContain("Casey");
    expect(screen.getByText("Dealer: Avery")).toBeTruthy();
  });

  it("submits cut size and tapped player order choices", () => {
    const onChooseDealing = vi.fn();
    const pendingDealChoice = {
      dealerIndex: 0,
      chooserIndex: 2,
      chooserPlayerId: "player-3",
    };

    const { rerender } = render(
      <GameLobbyView
        room={makeRoom({ pendingDealChoice })}
        game={null}
        completedGame={null}
        players={players}
        roomCode=""
        playerId="player-3"
        roomActions={{
          canCreateRoom: false,
          canJoinRoom: false,
          canLeaveRoom: true,
          canStartGame: false,
        }}
        dealChoice={{
          pendingDealChoice,
          dealChooserName: "Casey",
          isDealChooser: true,
        }}
        onRoomCodeChange={vi.fn()}
        onCreateRoom={vi.fn()}
        onJoinRoom={vi.fn()}
        onStartGame={vi.fn()}
        onChooseDealing={onChooseDealing}
        onLeaveRoom={vi.fn()}
        onCopyRoomCode={vi.fn()}
        onCopyRoomLink={vi.fn()}
        roomLink={null}
      />,
    );

    fireEvent.change(screen.getByLabelText("Cut size"), { target: { value: "7" } });
    fireEvent.click(screen.getByRole("button", { name: /Continue/ }));
    fireEvent.click(screen.getByRole("button", { name: "Start round" }));

    expect(onChooseDealing).toHaveBeenLastCalledWith({
      dealType: "round_robin",
      cutSize: 7,
    });

    rerender(
      <GameLobbyView
        room={makeRoom({ pendingDealChoice })}
        game={null}
        completedGame={null}
        players={players}
        roomCode=""
        playerId="player-3"
        roomActions={{
          canCreateRoom: false,
          canJoinRoom: false,
          canLeaveRoom: true,
          canStartGame: false,
        }}
        dealChoice={{
          pendingDealChoice,
          dealChooserName: "Casey",
          isDealChooser: true,
        }}
        onRoomCodeChange={vi.fn()}
        onCreateRoom={vi.fn()}
        onJoinRoom={vi.fn()}
        onStartGame={vi.fn()}
        onChooseDealing={onChooseDealing}
        onLeaveRoom={vi.fn()}
        onCopyRoomCode={vi.fn()}
        onCopyRoomLink={vi.fn()}
        roomLink={null}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    fireEvent.change(screen.getByLabelText("Cut size"), { target: { value: "4" } });
    fireEvent.click(screen.getByRole("button", { name: /Continue/ }));
    fireEvent.click(screen.getByRole("button", { name: "Tap order" }));
    act(() => {
      dndContextProps.onDragEnd?.({
        active: { id: "deal-player-1" },
        over: { id: "deal-player-0" },
      });
    });
    fireEvent.click(screen.getByRole("button", { name: "Start round" }));

    expect(onChooseDealing).toHaveBeenLastCalledWith({
      dealType: "tap",
      cutSize: 4,
      order: [2, 0, 1],
    });
  });
});

import { createContext, use, useEffect, useRef } from "react";
import { createStore } from "@tanstack/store";
import { useSelector } from "@tanstack/react-store";
import { getGameConnectionAuth } from "#/lib/game-auth";

type ConnectionStatus = "idle" | "disconnected" | "connecting" | "connected";

export type CardSnapshot = {
  rank?: number;
  suit?: number;
  isJoker?: boolean;
};

export type CompositionSnapshot = {
  type: string;
  cards: CardSnapshot[];
  jokerRepresentations?: Record<number, CardSnapshot[]>;
  points: number;
  complete: boolean;
};

export type CardActivitySnapshot = {
  kind: string;
  playerId: string;
};

export type CompositionActivitySnapshot = {
  tableIndex: number;
  kind?: string;
  playerId?: string;
  cardActivities?: Record<number, CardActivitySnapshot>;
};

export type DraftCompositionSnapshot = {
  tableIndex?: number;
  insertIndex?: number;
  cardInsertIndices?: Record<string, number>;
  reclaimTargets?: Record<string, number>;
  cards: CardSnapshot[];
};

export type TurnActivitySnapshot = {
  playerId: string;
  round: number;
  turnNumber: number;
  baselineCompositions?: CompositionSnapshot[];
  draftCompositions?: DraftCompositionSnapshot[];
  compositionActivities?: CompositionActivitySnapshot[];
};

export type CompositionDraftRequest = {
  cards: CardSnapshot[];
};

export type CompositionAdditionRequest = {
  compositionIndex: number;
  insertIndex?: number;
  cards: CardSnapshot[];
};

export type JokerReclaimRequest = {
  compositionIndex: number;
  jokerIndex: number;
  replacementCard: CardSnapshot;
};

export type TablePlayRequest = {
  compositions: CompositionDraftRequest[];
  additions: CompositionAdditionRequest[];
  reclaims: JokerReclaimRequest[];
};

export type TurnDraftUpdateRequest = {
  compositions: DraftCompositionSnapshot[];
};

export type PlayerStateSnapshot = {
  playerId: string;
  handCount: number;
  totalPoints: number;
  hasOpened: boolean;
};

export type TurnSnapshot = {
  number: number;
  playerIndex: number;
  playerId?: string;
  hasDrawn: boolean;
  mustUseDiscardDraw: boolean;
};

export type GameSnapshot = {
  phase: number;
  round: number;
  dealerIndex: number;
  roundWinnerIndex: number;
  turn: TurnSnapshot;
  players: PlayerStateSnapshot[];
  hand: CardSnapshot[];
  drawPileCount: number;
  discardPile: CardSnapshot[];
  activeCompositions: CompositionSnapshot[];
  turnActivity?: TurnActivitySnapshot;
};

export type PlayerSnapshot = {
  playerId: string;
  sessionId?: string;
  name: string;
  imageUrl?: string;
  connected: boolean;
  seat: number;
  isHost: boolean;
  canReconnect: boolean;
};

export type PendingDealChoiceSnapshot = {
  dealerIndex: number;
  chooserIndex: number;
  chooserPlayerId: string;
};

export type RoomSnapshot = {
  code: string;
  phase: string;
  hostPlayerId: string;
  dealerIndex?: number;
  pendingDealChoice?: PendingDealChoiceSnapshot;
  players: PlayerSnapshot[];
};

type ActionResult = {
  action: string;
  playerId: string;
  ok: boolean;
};

export type CompletedGameSnapshot = {
  room: RoomSnapshot;
  game: GameSnapshot;
};

export type LobbyState = {
  connectionStatus: ConnectionStatus;
  sessionId: string;
  playerId: string;
  room: RoomSnapshot | null;
  game: GameSnapshot | null;
  lastActionResult: ActionResult | null;
  lastError: string | null;
  lastErrorId: number;
  lastEvent: string | null;
  completedGame: CompletedGameSnapshot | null;
};

type Envelope<T = unknown> = {
  type: string;
  data?: T;
};

type ConnectionAuth = Awaited<ReturnType<typeof getGameConnectionAuth>>;

type GameWebSocketContextValue = {
  state: LobbyState;
  connect: () => Promise<void>;
  disconnect: () => void;
  createRoom: () => void;
  joinRoom: (roomCode: string) => void;
  startGame: () => void;
  startNextRound: () => void;
  chooseDealing: (dealType: string) => void;
  leaveRoom: () => void;
  drawFromDeck: () => void;
  drawFromDiscard: () => void;
  updateTurnDrafts: (draft: TurnDraftUpdateRequest) => void;
  playTable: (play: TablePlayRequest) => void;
  discardCard: (cardIndex: number) => void;
};

const initialState: LobbyState = {
  connectionStatus: "idle",
  sessionId: "",
  playerId: "",
  room: null,
  game: null,
  lastActionResult: null,
  lastError: null,
  lastErrorId: 0,
  lastEvent: null,
  completedGame: null,
};

const gameWebSocketStore = createStore(initialState);

const GameWebSocketContext = createContext<GameWebSocketContextValue | null>(null);

function withError(
  current: LobbyState,
  message: string,
  partial?: Partial<LobbyState>,
): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: message,
    lastErrorId: current.lastErrorId + 1,
  };
}

function clearError(current: LobbyState, partial?: Partial<LobbyState>): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: null,
  };
}

const incomingMessageReducers: Record<string, (current: LobbyState, data: any) => LobbyState> = {
  connected: (current, data) =>
    clearError(current, {
      connectionStatus: "connected",
      sessionId: data?.sessionId ?? current.sessionId,
      playerId: data?.playerId ?? current.playerId,
      lastEvent: "connected",
    }),
  room_state: (current, data) =>
    clearError(current, {
      room: data?.room ?? null,
      game: data?.room?.phase === "lobby" ? null : current.game,
      completedGame:
        data?.room?.phase === "lobby" && current.completedGame?.room.code === data?.room?.code
          ? current.completedGame
          : current.completedGame,
      lastEvent: "room_state",
    }),
  game_state: (current, data) =>
    clearError(current, {
      room: data?.room ?? current.room,
      game: data?.game ?? null,
      completedGame:
        data?.room?.phase === "game_over" && data?.game
          ? { room: data.room, game: data.game }
          : data?.room?.phase === "in_progress" || data?.room?.phase === "round_over"
            ? null
            : current.completedGame,
      lastEvent: "game_state",
    }),
  action_result: (current, data) =>
    clearError(current, {
      lastActionResult: data ?? null,
      lastEvent: "action_result",
    }),
  left_room: (current) =>
    clearError(current, {
      room: null,
      game: null,
      completedGame: null,
      lastEvent: "left_room",
    }),
  error: (current, data) =>
    withError(current, data?.message ?? "unknown error", {
      connectionStatus:
        current.connectionStatus === "connecting" ? "disconnected" : current.connectionStatus,
      lastEvent: "error",
    }),
};

function getDealerIndex(room: RoomSnapshot | null) {
  if (!room || room.players.length === 0) {
    return 0;
  }

  const hostIndex = room.players.findIndex((player) => player.isHost);
  return hostIndex >= 0 ? hostIndex : 0;
}

export function GameWebSocketProvider({ children }: { children: React.ReactNode }) {
  const socketRef = useRef<WebSocket | null>(null);
  const connectAttemptRef = useRef(0);
  const connectInFlightRef = useRef(false);
  const state = useSelector(gameWebSocketStore);

  useEffect(() => {
    function cancelPendingConnect() {
      connectAttemptRef.current += 1;
      connectInFlightRef.current = false;

      const socket = socketRef.current;
      socketRef.current = null;
      socket?.close();
    }

    window.addEventListener("beforeunload", cancelPendingConnect);
    window.addEventListener("pagehide", cancelPendingConnect);

    return () => {
      window.removeEventListener("beforeunload", cancelPendingConnect);
      window.removeEventListener("pagehide", cancelPendingConnect);
      cancelPendingConnect();
    };
  }, []);

  function updateState(updater: (current: LobbyState) => LobbyState) {
    gameWebSocketStore.setState((current) => updater(current));
  }

  function setConnectionError(message: string) {
    updateState((current) =>
      withError(current, message, {
        connectionStatus: "disconnected",
      }),
    );
  }

  function resolveConnectionAuthError(error: unknown) {
    if (error instanceof Error && error.message.trim() !== "") {
      return error.message;
    }
    return "failed to resolve websocket authentication";
  }

  function parseEnvelope(rawData: unknown): Envelope | null {
    if (typeof rawData !== "string") {
      return null;
    }

    try {
      const message = JSON.parse(rawData) as Envelope;
      if (typeof message?.type !== "string" || message.type === "") {
        return null;
      }
      return message;
    } catch {
      return null;
    }
  }

  function applyIncomingMessage(message: Envelope) {
    const reducer = incomingMessageReducers[message.type];
    if (!reducer) {
      updateState((current) => withError(current, `unknown websocket message: ${message.type}`));
      return;
    }

    updateState((current) => reducer(current, message.data));
  }

  async function getConnectionAuth(): Promise<ConnectionAuth> {
    try {
      return await getGameConnectionAuth();
    } catch (error) {
      setConnectionError(resolveConnectionAuthError(error));
      return null;
    }
  }

  function send(type: string, data: unknown) {
    const socket = socketRef.current;

    if (!socket || socket.readyState !== WebSocket.OPEN) {
      updateState((current) => withError(current, "websocket is not connected"));
      return;
    }

    socket.send(JSON.stringify({ type, data }));
    updateState((current) =>
      clearError(current, {
        lastEvent: type,
      }),
    );
  }

  async function connect() {
    const currentSocket = socketRef.current;

    if (connectInFlightRef.current) {
      return;
    }

    if (
      currentSocket &&
      (currentSocket.readyState === WebSocket.OPEN ||
        currentSocket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    const serverUrl = import.meta.env.VITE_GAME_SERVER_URL;

    if (!serverUrl) {
      setConnectionError("missing VITE_GAME_SERVER_URL");
      return;
    }

    const connectAttempt = connectAttemptRef.current + 1;
    connectAttemptRef.current = connectAttempt;
    connectInFlightRef.current = true;

    updateState((current) =>
      clearError(current, {
        connectionStatus: "connecting",
        lastEvent: null,
      }),
    );

    if (connectAttemptRef.current != connectAttempt) {
      connectInFlightRef.current = false;
      return;
    }

    const connectionAuth = await getConnectionAuth();
    const authToken =
      connectAttemptRef.current === connectAttempt ? connectionAuth?.authToken : null;

    if (!authToken) {
      connectInFlightRef.current = false;

      if (connectAttemptRef.current === connectAttempt) {
        setConnectionError("authentication required before connecting");
      }

      return;
    }

    const wsUrl = new URL("/ws", serverUrl).toString();

    const socket = new WebSocket(wsUrl);
    const sessionId = gameWebSocketStore.get().sessionId;

    if (connectAttemptRef.current !== connectAttempt) {
      socket.close();
      return;
    }

    socketRef.current = socket;

    socket.onopen = () => {
      if (connectAttemptRef.current !== connectAttempt || socketRef.current !== socket) {
        socket.close();
        return;
      }

      connectInFlightRef.current = false;
      socket.send(JSON.stringify({ type: "connect", data: { sessionId, authToken } }));
    };

    socket.onmessage = (event) => {
      const message = parseEnvelope(event.data);
      if (!message) {
        setConnectionError("received invalid websocket message");
        return;
      }

      applyIncomingMessage(message);
    };

    socket.onerror = () => {
      if (connectAttemptRef.current === connectAttempt) {
        connectInFlightRef.current = false;
      }
      setConnectionError("failed to connect websocket");
    };

    socket.onclose = () => {
      if (socketRef.current === socket) {
        socketRef.current = null;
        connectInFlightRef.current = false;

        updateState((current) => ({
          ...current,
          connectionStatus: "disconnected",
        }));
      }
    };
  }

  function disconnect() {
    connectAttemptRef.current += 1;
    connectInFlightRef.current = false;

    const socket = socketRef.current;
    socketRef.current = null;
    socket?.close();

    updateState((current) => ({
      ...current,
      room: null,
      game: null,
      lastEvent: "manual_disconnect",
    }));
  }

  const value: GameWebSocketContextValue = {
    state,
    connect,
    disconnect,
    createRoom: () => send("create_room", {}),
    joinRoom: (roomCode) => send("join_room", { roomCode }),
    startGame: () => send("start_game", { dealerIndex: getDealerIndex(state.room) }),
    startNextRound: () => send("start_next_round", {}),
    chooseDealing: (dealType) => send("choose_dealing", { dealType }),
    leaveRoom: () => send("leave_room", {}),
    drawFromDeck: () => send("draw", { source: "deck" }),
    drawFromDiscard: () => send("draw", { source: "discard" }),
    updateTurnDrafts: (draft) => send("draft_update", draft),
    playTable: (play) => send("play", play),
    discardCard: (cardIndex) => send("discard", { cardIndex }),
  };

  return <GameWebSocketContext value={value}>{children}</GameWebSocketContext>;
}

export function useGameWebSocket() {
  const context = use(GameWebSocketContext);

  if (!context) {
    throw new Error("useGameWebSocket must be used within a GameWebSocketProvider");
  }

  return context;
}

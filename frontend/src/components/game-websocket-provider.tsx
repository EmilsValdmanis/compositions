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

export type DealingChoiceRequest = {
  dealType: string;
  cutSize: number;
  order?: number[];
};

export type TurnDraftUpdateRequest = {
  compositions: DraftCompositionSnapshot[];
};

export type PlayerStateSnapshot = {
  playerId: string;
  handCount: number;
  hand?: CardSnapshot[];
  totalPoints: number;
  pointsGained: number;
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
  activeEmote?: {
    id: string;
    emoji: string;
    expiresAt: string;
  };
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

export type ActionResult = {
  action: string;
  playerId: string;
  ok: boolean;
};

type PendingAction = {
  expectedAction: string;
  id: number;
  resolve: (result: ActionResult) => void;
  reject: (error: Error) => void;
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

const reconnectBaseDelayMs = 500;
const reconnectMaxDelayMs = 5_000;

type GameWebSocketContextValue = {
  state: LobbyState;
  connect: () => Promise<void>;
  disconnect: () => void;
  createRoom: () => void;
  joinRoom: (roomCode: string) => void;
  startGame: () => void;
  startNextRound: () => void;
  chooseDealing: (choice: DealingChoiceRequest | string) => void;
  leaveRoom: () => void;
  sendEmote: (emoji: string) => void;
  drawFromDeck: () => void;
  drawFromDiscard: () => void;
  updateTurnDrafts: (draft: TurnDraftUpdateRequest) => void;
  playTable: (play: TablePlayRequest) => Promise<ActionResult>;
  discardCard: (cardIndex: number) => Promise<ActionResult>;
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

function isRejectedSessionError(message: string) {
  return message === "session not found" || message === "session belongs to a different user";
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
  error: (current, data) => {
    const message = data?.message ?? "unknown error";
    if (current.lastEvent === "draft_update" && message === "not your turn") {
      return clearError(current, {
        lastEvent: "error",
      });
    }

    return withError(current, message, {
      connectionStatus:
        current.connectionStatus === "connecting" ? "disconnected" : current.connectionStatus,
      sessionId: isRejectedSessionError(message) ? "" : current.sessionId,
      lastEvent: "error",
    });
  },
};

function getDealerIndex(room: RoomSnapshot | null) {
  if (!room || room.players.length === 0) {
    return 0;
  }

  const hostIndex = room.players.findIndex((player) => player.isHost);
  return hostIndex >= 0 ? hostIndex : 0;
}

function useGameWebSocketController(): GameWebSocketContextValue {
  const socketRef = useRef<WebSocket | null>(null);
  const connectAttemptRef = useRef(0);
  const connectInFlightRef = useRef(false);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectAttemptRef = useRef(0);
  const nextPendingActionIdRef = useRef(0);
  const pendingActionsRef = useRef<PendingAction[]>([]);
  const state = useSelector(gameWebSocketStore);

  useEffect(() => {
    function clearReconnectTimer() {
      if (reconnectTimerRef.current) {
        window.clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    }

    function cancelPendingConnect() {
      clearReconnectTimer();
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

  function rejectPendingActions(message: string) {
    const pendingActions = pendingActionsRef.current.splice(0);

    for (const pendingAction of pendingActions) {
      pendingAction.reject(new Error(message));
    }
  }

  function resolvePendingAction(data: unknown) {
    const result = data as Partial<ActionResult> | null | undefined;
    const action = typeof result?.action === "string" ? result.action : null;
    const pendingIndex = action
      ? pendingActionsRef.current.findIndex(
          (pendingAction) => pendingAction.expectedAction === action,
        )
      : 0;
    const [pendingAction] = pendingActionsRef.current.splice(
      pendingIndex >= 0 ? pendingIndex : 0,
      1,
    );

    if (!pendingAction) {
      return;
    }

    if (result?.ok === false) {
      pendingAction.reject(new Error(`${action ?? pendingAction.expectedAction} failed`));
      return;
    }

    pendingAction.resolve({
      action: action ?? pendingAction.expectedAction,
      playerId: result?.playerId ?? "",
      ok: true,
    });
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
    if (message.type === "action_result") {
      resolvePendingAction(message.data);
    }

    if (message.type === "error") {
      rejectPendingActions(
        typeof message.data === "object" && message.data && "message" in message.data
          ? String(message.data.message)
          : "unknown error",
      );
    }

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

  function send(type: string, data: unknown): void;
  function send(type: string, data: unknown, options: { awaitResult: true }): Promise<ActionResult>;
  function send(type: string, data: unknown, options?: { awaitResult?: boolean }) {
    const socket = socketRef.current;

    if (!socket || socket.readyState !== WebSocket.OPEN) {
      updateState((current) => withError(current, "websocket is not connected"));

      if (options?.awaitResult) {
        return Promise.reject(new Error("websocket is not connected"));
      }

      return;
    }

    const pendingAction = options?.awaitResult
      ? new Promise<ActionResult>((resolve, reject) => {
          const pendingActionId = nextPendingActionIdRef.current;
          nextPendingActionIdRef.current += 1;

          pendingActionsRef.current.push({
            expectedAction: type,
            id: pendingActionId,
            resolve,
            reject,
          });
        })
      : undefined;

    try {
      socket.send(JSON.stringify({ type, data }));
    } catch (error) {
      if (options?.awaitResult) {
        const pendingIndex = pendingActionsRef.current.findLastIndex(
          (candidate) => candidate.expectedAction === type,
        );
        const [queuedAction] = pendingActionsRef.current.splice(
          pendingIndex >= 0 ? pendingIndex : 0,
          1,
        );
        queuedAction?.reject(error instanceof Error ? error : new Error("failed to send action"));
        return pendingAction;
      }

      updateState((current) => withError(current, "failed to send websocket action"));
      return;
    }

    updateState((current) =>
      clearError(current, {
        lastEvent: type,
      }),
    );

    return pendingAction;
  }

  async function connect() {
    if (reconnectTimerRef.current) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }

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

    return getConnectionAuth().then((connectionAuth) => {
      if (connectAttemptRef.current !== connectAttempt) {
        connectInFlightRef.current = false;
        return;
      }

      if (!connectionAuth) {
        connectInFlightRef.current = false;
        setConnectionError("authentication required before connecting");
        return;
      }

      const wsUrl = new URL("/ws", serverUrl).toString();

      const socket = new WebSocket(wsUrl);

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
        reconnectAttemptRef.current = 0;
        socket.send(
          JSON.stringify({
            type: "connect",
            data: { sessionId: gameWebSocketStore.get().sessionId },
          }),
        );
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
        rejectPendingActions("failed to connect websocket");
        setConnectionError("failed to connect websocket");
      };

      socket.onclose = () => {
        if (socketRef.current === socket) {
          socketRef.current = null;
          connectInFlightRef.current = false;
          rejectPendingActions("websocket disconnected");

          updateState((current) => ({
            ...current,
            connectionStatus: "disconnected",
          }));

          const reconnectAttempt = reconnectAttemptRef.current;
          reconnectAttemptRef.current += 1;
          const reconnectDelay = Math.min(
            reconnectBaseDelayMs * 2 ** reconnectAttempt,
            reconnectMaxDelayMs,
          );
          reconnectTimerRef.current = window.setTimeout(() => {
            reconnectTimerRef.current = null;
            void connect();
          }, reconnectDelay);
        }
      };
    });
  }

  function disconnect() {
    if (reconnectTimerRef.current) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    reconnectAttemptRef.current = 0;
    connectAttemptRef.current += 1;
    connectInFlightRef.current = false;

    const socket = socketRef.current;
    socketRef.current = null;
    socket?.close();
    rejectPendingActions("websocket disconnected");

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
    chooseDealing: (choice) =>
      send("choose_dealing", typeof choice === "string" ? { dealType: choice } : choice),
    leaveRoom: () => send("leave_room", {}),
    sendEmote: (emoji) => send("send_emote", { emoji }),
    drawFromDeck: () => send("draw", { source: "deck" }),
    drawFromDiscard: () => send("draw", { source: "discard" }),
    updateTurnDrafts: (draft) => send("draft_update", draft),
    playTable: (play) => send("play", play, { awaitResult: true }),
    discardCard: (cardIndex) => send("discard", { cardIndex }, { awaitResult: true }),
  };

  return value;
}

export function GameWebSocketProvider({ children }: { children: React.ReactNode }) {
  const value = useGameWebSocketController();
  return <GameWebSocketContext value={value}>{children}</GameWebSocketContext>;
}

export function useGameWebSocket() {
  const context = use(GameWebSocketContext);

  if (!context) {
    throw new Error("useGameWebSocket must be used within a GameWebSocketProvider");
  }

  return context;
}

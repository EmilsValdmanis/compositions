import { createContext, use, useEffect, useRef } from "react";
import { createStore } from "@tanstack/store";
import { useSelector } from "@tanstack/react-store";
import { getGameConnectionAuth } from "#/lib/game-auth";
import { gameErrorMessage } from "#/lib/game-error-messages";

type ConnectionStatus = "idle" | "disconnected" | "connecting" | "connected";

export type GameMode = "quick" | "full";

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
  id: string;
  tableIndex?: number;
  insertIndex?: number;
  cardPlacements?: Array<{
    insertIndex?: number;
    reclaimJokerIndex?: number;
  }>;
  cards: CardSnapshot[];
};

export type TurnActivitySnapshot = {
  playerId: string;
  round: number;
  turnNumber: number;
  drawSource?: "deck" | "discard";
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
  unadjustedTotalPoints?: number;
  hasOpened: boolean;
  forfeited?: boolean;
};

export type TurnSnapshot = {
  number: number;
  playerIndex: number;
  playerId?: string;
  hasDrawn: boolean;
  mustUseDiscardDraw: boolean;
};

export type GameSnapshot = {
  gameMode?: GameMode;
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
  userId?: string;
  sessionId?: string;
  name: string;
  imageUrl?: string;
  connected: boolean;
  seat: number;
  isHost: boolean;
  canReconnect: boolean;
  forfeited?: boolean;
  activeEmote?: {
    id: string;
    emoji: string;
    expiresAt: string;
  };
};

export type EndGameProposalSnapshot = {
  id: string;
  kind: "mutual_end" | "technical_abort";
  proposerPlayerId: string;
  description?: string;
  eligiblePlayerIds: string[];
  agreedPlayerIds: string[];
  expiresAt: string;
};

export type GameConclusionSnapshot = {
  kind: "forfeit" | "mutual_end" | "technical_abort";
  winnerPlayerId?: string;
  reportId?: string;
};

export type PendingDealChoiceSnapshot = {
  dealerIndex: number;
  chooserIndex: number;
  chooserPlayerId: string;
};

export type RoomSnapshot = {
  code: string;
  phase: string;
  gameMode?: GameMode;
  hostPlayerId: string;
  dealerIndex?: number;
  pendingDealChoice?: PendingDealChoiceSnapshot;
  endProposal?: EndGameProposalSnapshot;
  conclusion?: GameConclusionSnapshot;
  players: PlayerSnapshot[];
};

export type ActionResult = {
  action: string;
  playerId: string;
  ok: boolean;
};

export type SocialUser = {
  id: string;
  name: string;
  imageUrl?: string;
  online: boolean;
};

export type FriendRequest = {
  id: string;
  user: SocialUser;
  createdAt: string;
};

export type GameInvite = {
  id: string;
  user: SocialUser;
  roomCode: string;
  createdAt: string;
  expiresAt: string;
};

export type SocialState = {
  userId: string;
  friends: SocialUser[];
  incomingFriendRequests: FriendRequest[];
  outgoingFriendRequestUserIds: string[];
  gameInvites: GameInvite[];
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
  lastErrorCode: string | null;
  lastErrorMessage: string | null;
  lastErrorId: number;
  lastEvent: string | null;
  completedGame: CompletedGameSnapshot | null;
  social: SocialState;
};

type Envelope<T = unknown> = {
  type: string;
  data?: T;
};

type ConnectionAuth = Awaited<ReturnType<typeof getGameConnectionAuth>>;

const reconnectBaseDelayMs = 500;
const reconnectMaxDelayMs = 5_000;

function emptySocialState(): SocialState {
  return {
    userId: "",
    friends: [],
    incomingFriendRequests: [],
    outgoingFriendRequestUserIds: [],
    gameInvites: [],
  };
}

type GameWebSocketContextValue = {
  state: LobbyState;
  connect: () => Promise<void>;
  disconnect: () => void;
  dismissCompletedGame: () => void;
  createRoom: () => void;
  joinRoom: (roomCode: string) => void;
  sendFriendRequest: (userId: string) => Promise<ActionResult>;
  respondFriendRequest: (requestId: string, accept: boolean) => Promise<ActionResult>;
  removeFriend: (userId: string) => Promise<ActionResult>;
  sendGameInvite: (userId: string) => Promise<ActionResult>;
  respondGameInvite: (inviteId: string, accept: boolean) => Promise<ActionResult>;
  startGame: (gameMode: GameMode) => void;
  startNextRound: () => void;
  chooseDealing: (choice: DealingChoiceRequest | string) => void;
  leaveRoom: () => void;
  forfeitGame: () => Promise<ActionResult>;
  requestEndGame: () => Promise<ActionResult>;
  voteEndGame: (proposalId: string, approve: boolean) => Promise<ActionResult>;
  reportIssue: (description: string, requestAbort: boolean) => Promise<ActionResult>;
  sendEmote: (emoji: string) => void;
  drawFromDeck: () => void;
  drawFromDiscard: () => void;
  updateTurnDrafts: (draft: TurnDraftUpdateRequest) => void;
  playTable: (play: TablePlayRequest) => Promise<ActionResult>;
  playTableAndDiscard: (
    play: TablePlayRequest,
    cardIndex: number,
    card: CardSnapshot,
  ) => Promise<ActionResult>;
  discardCard: (cardIndex: number, card: CardSnapshot) => Promise<ActionResult>;
};

const initialState: LobbyState = {
  connectionStatus: "idle",
  sessionId: "",
  playerId: "",
  room: null,
  game: null,
  lastActionResult: null,
  lastError: null,
  lastErrorCode: null,
  lastErrorMessage: null,
  lastErrorId: 0,
  lastEvent: null,
  completedGame: null,
  social: emptySocialState(),
};

const gameWebSocketStore = createStore(initialState);

const GameWebSocketContext = createContext<GameWebSocketContextValue | null>(null);

function withError(
  current: LobbyState,
  code: string,
  partial?: Partial<LobbyState>,
  developerMessage = code,
): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: gameErrorMessage(code),
    lastErrorCode: code,
    lastErrorMessage: developerMessage,
    lastErrorId: current.lastErrorId + 1,
  };
}

function clearError(current: LobbyState, partial?: Partial<LobbyState>): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: null,
    lastErrorCode: null,
    lastErrorMessage: null,
  };
}

function isRejectedSessionError(code: string) {
  return code === "session_not_found" || code === "session_user_mismatch";
}

function isSocialAction(action: unknown) {
  return (
    action === "send_friend_request" ||
    action === "respond_friend_request" ||
    action === "remove_friend" ||
    action === "send_game_invite" ||
    action === "respond_game_invite"
  );
}

const incomingMessageReducers = new Map<string, (current: LobbyState, data: any) => LobbyState>(
  Object.entries({
    connected: (current, data) =>
      clearError(current, {
        connectionStatus: "connected",
        sessionId: data?.sessionId ?? current.sessionId,
        playerId: data?.playerId ?? current.playerId,
        lastEvent: "connected",
      }),
    room_state: (current, data) => {
      const room = data?.room ?? null;
      const completedGame =
        room?.phase === "game_over" && current.game
          ? { room, game: current.game }
          : current.completedGame;

      return clearError(current, {
        room,
        game: room?.phase === "lobby" ? null : current.game,
        completedGame,
        lastEvent: "room_state",
      });
    },
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
    social_state: (current, data) =>
      clearError(current, {
        social: {
          userId: data?.userId ?? current.social.userId,
          friends: Array.isArray(data?.friends) ? data.friends : [],
          incomingFriendRequests: Array.isArray(data?.incomingFriendRequests)
            ? data.incomingFriendRequests
            : [],
          outgoingFriendRequestUserIds: Array.isArray(data?.outgoingFriendRequestUserIds)
            ? data.outgoingFriendRequestUserIds
            : [],
          gameInvites: Array.isArray(data?.gameInvites) ? data.gameInvites : [],
        },
        lastEvent: "social_state",
      }),
    error: (current, data) => {
      const code = data?.code ?? "internal_error";
      const developerMessage =
        typeof data?.message === "string" && data.message !== "" ? data.message : code;
      if (data?.action === "draft_update" || isSocialAction(data?.action)) {
        return clearError(current, {
          lastEvent: "error",
        });
      }

      return withError(
        current,
        code,
        {
          connectionStatus:
            current.connectionStatus === "connecting" ? "disconnected" : current.connectionStatus,
          sessionId: isRejectedSessionError(code) ? "" : current.sessionId,
          lastEvent: "error",
        },
        developerMessage,
      );
    },
  }),
);

function getDealerIndex(room: RoomSnapshot | null) {
  if (!room || room.players.length === 0) {
    return 0;
  }

  const hostIndex = room.players.findIndex((player) => player.isHost);
  return hostIndex >= 0 ? hostIndex : 0;
}

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

function resolveConnectionAuthError(_error: unknown) {
  return "auth_required";
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

async function getConnectionAuth(): Promise<ConnectionAuth> {
  try {
    return await getGameConnectionAuth();
  } catch (error) {
    setConnectionError(resolveConnectionAuthError(error));
    return null;
  }
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

  function rejectPendingActions(message: string, action?: string) {
    if (action) {
      const pendingIndex = pendingActionsRef.current.findIndex(
        (pendingAction) => pendingAction.expectedAction === action,
      );
      if (pendingIndex < 0) {
        return;
      }

      const [pendingAction] = pendingActionsRef.current.splice(pendingIndex, 1);
      pendingAction?.reject(new Error(message));
      return;
    }

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
      pendingAction.reject(new Error("send_failed"));
      return;
    }

    pendingAction.resolve({
      action: action ?? pendingAction.expectedAction,
      playerId: result?.playerId ?? "",
      ok: true,
    });
  }

  function applyIncomingMessage(message: Envelope) {
    if (message.type === "action_result") {
      resolvePendingAction(message.data);
    }

    if (message.type === "error") {
      const action =
        typeof message.data === "object" &&
        message.data &&
        "action" in message.data &&
        typeof message.data.action === "string"
          ? message.data.action
          : undefined;
      rejectPendingActions(
        typeof message.data === "object" && message.data && "message" in message.data
          ? String(message.data.message)
          : typeof message.data === "object" && message.data && "code" in message.data
            ? String(message.data.code)
            : "internal_error",
        action,
      );
    }

    const reducer = incomingMessageReducers.get(message.type);
    if (!reducer) {
      updateState((current) => withError(current, "unknown_message_type"));
      return;
    }

    updateState((current) => reducer(current, message.data));
  }

  function send(type: string, data: unknown): void;
  function send(type: string, data: unknown, options: { awaitResult: true }): Promise<ActionResult>;
  function send(type: string, data: unknown, options?: { awaitResult?: boolean }) {
    const socket = socketRef.current;

    if (!socket || socket.readyState !== WebSocket.OPEN) {
      updateState((current) => withError(current, "connection_unavailable"));

      if (options?.awaitResult) {
        return Promise.reject(new Error("connection_unavailable"));
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
        queuedAction?.reject(error instanceof Error ? error : new Error("send_failed"));
        return pendingAction;
      }

      updateState((current) => withError(current, "send_failed"));
      return;
    }

    updateState((current) =>
      clearError(current, {
        lastEvent: type,
      }),
    );

    return pendingAction;
  }

  const connect = async function connectWebSocket() {
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
      setConnectionError("internal_error");
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
        setConnectionError("auth_required");
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
          setConnectionError("invalid_server_message");
          return;
        }

        applyIncomingMessage(message);
      };

      socket.onerror = () => {
        if (connectAttemptRef.current === connectAttempt) {
          connectInFlightRef.current = false;
        }
        rejectPendingActions("connection_failed");
        setConnectionError("connection_failed");
      };

      socket.onclose = () => {
        if (socketRef.current === socket) {
          socketRef.current = null;
          connectInFlightRef.current = false;
          rejectPendingActions("connection_lost");

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
            void connectWebSocket();
          }, reconnectDelay);
        }
      };
    });
  };

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
    rejectPendingActions("connection_lost");

    updateState((current) => ({
      ...current,
      connectionStatus: "disconnected",
      room: null,
      game: null,
      social: emptySocialState(),
      lastEvent: "manual_disconnect",
    }));
  }

  const value: GameWebSocketContextValue = {
    state,
    connect,
    disconnect,
    dismissCompletedGame: () =>
      updateState((current) => ({
        ...current,
        completedGame: null,
      })),
    createRoom: () => send("create_room", {}),
    joinRoom: (roomCode) => send("join_room", { roomCode }),
    sendFriendRequest: (userId) => send("send_friend_request", { userId }, { awaitResult: true }),
    respondFriendRequest: (requestId, accept) =>
      send("respond_friend_request", { requestId, accept }, { awaitResult: true }),
    removeFriend: (userId) => send("remove_friend", { userId }, { awaitResult: true }),
    sendGameInvite: (userId) => send("send_game_invite", { userId }, { awaitResult: true }),
    respondGameInvite: (inviteId, accept) =>
      send("respond_game_invite", { inviteId, accept }, { awaitResult: true }),
    startGame: (gameMode) =>
      send("start_game", { dealerIndex: getDealerIndex(state.room), gameMode }),
    startNextRound: () => send("start_next_round", {}),
    chooseDealing: (choice) =>
      send("choose_dealing", typeof choice === "string" ? { dealType: choice } : choice),
    leaveRoom: () => send("leave_room", {}),
    forfeitGame: () => send("forfeit_game", {}, { awaitResult: true }),
    requestEndGame: () => send("request_end_game", { kind: "mutual_end" }, { awaitResult: true }),
    voteEndGame: (proposalId, approve) =>
      send("vote_end_game", { proposalId, approve }, { awaitResult: true }),
    reportIssue: (description, requestAbort) =>
      send("report_issue", { description, requestAbort }, { awaitResult: true }),
    sendEmote: (emoji) => send("send_emote", { emoji }),
    drawFromDeck: () => send("draw", { source: "deck" }),
    drawFromDiscard: () => send("draw", { source: "discard" }),
    updateTurnDrafts: (draft) => send("draft_update", draft),
    playTable: (play) => send("play", play, { awaitResult: true }),
    playTableAndDiscard: (play, cardIndex, card) =>
      send("play_and_discard", { ...play, cardIndex, card }, { awaitResult: true }),
    discardCard: (cardIndex, card) => send("discard", { cardIndex, card }, { awaitResult: true }),
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

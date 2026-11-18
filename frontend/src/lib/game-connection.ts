import { createStore } from "@tanstack/store";
import { hasGameSession } from "#/lib/game-auth";
import {
  isSocialAction,
  parseGameEvent,
  type AcknowledgedAction,
  type ActionResult,
  type CardSnapshot,
  type DealingChoiceRequest,
  type GameMode,
  type RoomSnapshot,
  type TablePlayRequest,
  type TurnDraftUpdateRequest,
} from "#/lib/game-protocol";
import {
  clearError,
  emptySocialState,
  initialLobbyState,
  reduceIncomingMessage,
  withError,
} from "#/lib/game-lobby-state";

type GameCommands = {
  connect: () => Promise<void>;
  disconnect: () => void;
  dismissError: () => void;
  dismissCompletedGame: () => void;
  createRoom: () => void;
  joinRoom: (roomCode: string) => void;
  sendFriendRequest: (userId: string) => Promise<ActionResult>;
  respondFriendRequest: (requestId: string, accept: boolean) => Promise<ActionResult>;
  removeFriend: (userId: string) => Promise<ActionResult>;
  sendGameInvite: (userId: string) => Promise<ActionResult>;
  respondGameInvite: (inviteId: string, accept: boolean) => Promise<ActionResult>;
  spectateGame: (userId: string) => Promise<ActionResult>;
  stopSpectating: () => Promise<ActionResult>;
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

function getDealerIndex(room: RoomSnapshot | null) {
  if (!room || room.players.length === 0) {
    return 0;
  }

  const hostIndex = room.players.findIndex((player) => player.isHost);
  return hostIndex >= 0 ? hostIndex : 0;
}

type PendingAction = {
  action: AcknowledgedAction;
  playerId?: string;
  timer: number;
  resolve: (result: ActionResult) => void;
  reject: (error: Error) => void;
};

const reconnectBaseDelayMs = 500;
const reconnectMaxDelayMs = 5_000;
const replyTimeoutMs = 20_000;

export function createGameConnection() {
  const store = createStore(initialLobbyState());
  const pendingActions = new Map<string, PendingAction>();
  let socket: WebSocket | null = null;
  let authAttempt: symbol | undefined;
  let reconnectTimer: number | undefined;
  let reconnectAttempt = 0;

  function takePendingAction(requestId: string) {
    const pending = pendingActions.get(requestId);
    if (pending) {
      pendingActions.delete(requestId);
      window.clearTimeout(pending.timer);
    }
    return pending;
  }

  function rejectPendingAction(requestId: string, message: string) {
    takePendingAction(requestId)?.reject(new Error(message));
  }

  function rejectPendingActions(message: string) {
    for (const requestId of pendingActions.keys()) {
      rejectPendingAction(requestId, message);
    }
  }

  function resolveReply(result: ActionResult | undefined) {
    if (!result?.requestId) return;
    const pending = pendingActions.get(result.requestId);
    if (!pending || result.action !== pending.action) return;
    if (pending.playerId && result.playerId !== pending.playerId) return;

    takePendingAction(result.requestId);
    if (result.ok === true) {
      pending.resolve(result);
    } else {
      pending.reject(new Error("send_failed"));
    }
  }

  function send(type: string, data: unknown, requestId?: string) {
    if (socket?.readyState !== WebSocket.OPEN) {
      store.setState((state) => withError(state, "connection_unavailable"));
      return "connection_unavailable";
    }

    try {
      socket.send(JSON.stringify({ type, data, requestId }));
      store.setState((state) => clearError(state));
    } catch (error) {
      if (!requestId) store.setState((state) => withError(state, "send_failed"));
      return error instanceof Error && error.message ? error.message : "send_failed";
    }
  }

  function sendForResult(action: AcknowledgedAction, data: unknown) {
    return new Promise<ActionResult>((resolve, reject) => {
      const requestId = crypto.randomUUID();
      pendingActions.set(requestId, {
        action,
        playerId: isSocialAction(action) ? undefined : store.get().playerId,
        timer: window.setTimeout(
          () => rejectPendingAction(requestId, "command_timeout"),
          replyTimeoutMs,
        ),
        resolve,
        reject,
      });
      const error = send(action, data, requestId);
      if (error) rejectPendingAction(requestId, error);
    });
  }

  function clearReconnectTimer() {
    if (reconnectTimer === undefined) return;
    window.clearTimeout(reconnectTimer);
    reconnectTimer = undefined;
  }

  function scheduleReconnect() {
    if (reconnectTimer !== undefined) return;
    const delay = Math.min(reconnectBaseDelayMs * 2 ** reconnectAttempt, reconnectMaxDelayMs);
    reconnectAttempt += 1;
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = undefined;
      void connect();
    }, delay);
  }

  function setConnectionError(code: string) {
    store.setState((state) => withError(state, code, { connectionStatus: "disconnected" }));
  }

  function markDisconnected() {
    store.setState((state) => ({ ...state, connectionStatus: "disconnected" }));
  }

  async function connect() {
    clearReconnectTimer();
    if (authAttempt || socket) return;

    const serverURL = import.meta.env.VITE_GAME_SERVER_URL;
    if (!serverURL) {
      setConnectionError("internal_error");
      return;
    }

    const attempt = Symbol();
    authAttempt = attempt;
    store.setState((state) =>
      clearError(state, {
        connectionStatus: state.connectionStatus === "disconnected" ? "disconnected" : "connecting",
      }),
    );

    try {
      const authenticated = await hasGameSession();
      if (authAttempt !== attempt) return;
      if (!authenticated) {
        setConnectionError("auth_required");
        return;
      }
    } catch {
      if (authAttempt === attempt) {
        markDisconnected();
        scheduleReconnect();
      }
      return;
    } finally {
      if (authAttempt === attempt) authAttempt = undefined;
    }

    let current: WebSocket;
    try {
      current = new WebSocket(new URL("/ws", serverURL).toString());
    } catch {
      setConnectionError("internal_error");
      return;
    }
    socket = current;

    function handleConnectionLoss(error: string) {
      if (socket !== current) return;
      socket = null;
      current.close();
      rejectPendingActions(error);
      markDisconnected();
      scheduleReconnect();
    }

    current.onopen = () => {
      if (socket !== current) return;
      try {
        current.send(
          JSON.stringify({ type: "connect", data: { sessionId: store.get().sessionId } }),
        );
      } catch {
        handleConnectionLoss("connection_failed");
      }
    };

    current.onmessage = ({ data }) => {
      if (socket !== current) return;
      const event = parseGameEvent(data);
      if (!event) {
        setConnectionError("invalid_server_message");
        return;
      }

      switch (event.type) {
        case "connected":
          reconnectAttempt = 0;
          break;
        case "action_result":
          resolveReply(event.data);
          break;
        case "error": {
          const { requestId, action, message, code } = event.data ?? {};
          if (requestId && pendingActions.get(requestId)?.action === action) {
            rejectPendingAction(requestId, message ?? code ?? "internal_error");
          }
          break;
        }
      }
      store.setState((state) => reduceIncomingMessage(state, event));
    };

    current.onerror = () => handleConnectionLoss("connection_failed");
    current.onclose = () => handleConnectionLoss("connection_lost");
  }

  function stop() {
    clearReconnectTimer();
    reconnectAttempt = 0;
    authAttempt = undefined;
    const previous = socket;
    socket = null;
    previous?.close();
    rejectPendingActions("connection_lost");
  }

  function disconnect() {
    stop();
    store.setState((state) => ({
      ...state,
      connectionStatus: "disconnected",
      room: null,
      game: null,
      isSpectating: false,
      social: emptySocialState(),
    }));
  }

  const commands: GameCommands = {
    connect,
    disconnect,
    dismissError: () => store.setState((state) => clearError(state)),
    dismissCompletedGame: () => store.setState((state) => ({ ...state, completedGame: null })),
    createRoom: () => send("create_room", {}),
    joinRoom: (roomCode) => send("join_room", { roomCode }),
    sendFriendRequest: (userId) => sendForResult("send_friend_request", { userId }),
    respondFriendRequest: (requestId, accept) =>
      sendForResult("respond_friend_request", { requestId, accept }),
    removeFriend: (userId) => sendForResult("remove_friend", { userId }),
    sendGameInvite: (userId) => sendForResult("send_game_invite", { userId }),
    respondGameInvite: (inviteId, accept) =>
      sendForResult("respond_game_invite", { inviteId, accept }),
    spectateGame: (userId) => sendForResult("spectate_game", { userId }),
    stopSpectating: () => sendForResult("stop_spectating", {}),
    startGame: (gameMode) =>
      send("start_game", { dealerIndex: getDealerIndex(store.get().room), gameMode }),
    startNextRound: () => send("start_next_round", {}),
    chooseDealing: (choice) =>
      send("choose_dealing", typeof choice === "string" ? { dealType: choice } : choice),
    leaveRoom: () => send("leave_room", {}),
    forfeitGame: () => sendForResult("forfeit_game", {}),
    requestEndGame: () => sendForResult("request_end_game", { kind: "mutual_end" }),
    voteEndGame: (proposalId, approve) => sendForResult("vote_end_game", { proposalId, approve }),
    reportIssue: (description, requestAbort) =>
      sendForResult("report_issue", { description, requestAbort }),
    sendEmote: (emoji) => send("send_emote", { emoji }),
    drawFromDeck: () => send("draw", { source: "deck" }),
    drawFromDiscard: () => send("draw", { source: "discard" }),
    updateTurnDrafts: (draft) => send("draft_update", draft),
    playTable: (play) => sendForResult("play", play),
    playTableAndDiscard: (play, cardIndex, card) =>
      sendForResult("play_and_discard", { ...play, cardIndex, card }),
    discardCard: (cardIndex, card) => sendForResult("discard", { cardIndex, card }),
  };

  return { store, commands, stop };
}

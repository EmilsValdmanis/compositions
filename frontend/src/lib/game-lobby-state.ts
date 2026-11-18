import {
  isSocialAction,
  type GameEvent,
  type GameSnapshot,
  type RoomSnapshot,
  type SocialState,
} from "#/lib/game-protocol";
import { gameErrorMessage } from "#/lib/game-error-messages";

type ConnectionStatus = "idle" | "disconnected" | "connecting" | "connected";

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
  lastError: string | null;
  lastErrorCode: string | null;
  lastErrorId: number;
  completedGame: CompletedGameSnapshot | null;
  isSpectating: boolean;
  social: SocialState;
};

export function emptySocialState(): SocialState {
  return {
    userId: "",
    friends: [],
    incomingFriendRequests: [],
    outgoingFriendRequestUserIds: [],
    gameInvites: [],
  };
}

export function initialLobbyState(): LobbyState {
  return {
    connectionStatus: "idle",
    sessionId: "",
    playerId: "",
    room: null,
    game: null,
    lastError: null,
    lastErrorCode: null,
    lastErrorId: 0,
    completedGame: null,
    isSpectating: false,
    social: emptySocialState(),
  };
}

export function withError(
  current: LobbyState,
  code: string,
  partial?: Partial<LobbyState>,
): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: gameErrorMessage(code),
    lastErrorCode: code,
    lastErrorId: current.lastErrorId + 1,
  };
}

export function clearError(current: LobbyState, partial?: Partial<LobbyState>): LobbyState {
  return {
    ...current,
    ...partial,
    lastError: null,
    lastErrorCode: null,
  };
}

function isRejectedSessionError(code: string) {
  return code === "session_not_found" || code === "session_user_mismatch";
}

export function reduceIncomingMessage(current: LobbyState, event: GameEvent): LobbyState {
  const { type, data } = event;
  switch (type) {
    case "connected":
      return clearError(current, {
        connectionStatus: "connected",
        sessionId: data?.sessionId ?? current.sessionId,
        playerId: data?.playerId ?? current.playerId,
        isSpectating: false,
      });
    case "room_state": {
      const room = data?.room ?? null;
      const completedGame =
        room?.phase === "game_over" && current.game
          ? { room, game: current.game }
          : current.completedGame;

      return clearError(current, {
        room,
        game: room?.phase === "lobby" ? null : current.game,
        completedGame,
      });
    }
    case "game_state":
      return clearError(current, {
        room: data?.room ?? current.room,
        game: data?.game ?? null,
        isSpectating: data?.spectating === true,
        completedGame:
          data?.room?.phase === "game_over" && data?.game
            ? { room: data.room, game: data.game }
            : data?.room?.phase === "in_progress" || data?.room?.phase === "round_over"
              ? null
              : current.completedGame,
      });
    case "spectating_ended":
      return clearError(current, {
        room: null,
        game: null,
        isSpectating: false,
      });
    case "action_result":
      return clearError(current);
    case "left_room":
      return clearError(current, {
        room: null,
        game: null,
        completedGame: null,
        isSpectating: false,
      });
    case "social_state":
      return clearError(current, {
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
      });
    case "error": {
      const code = data?.code ?? "internal_error";
      if (data?.action === "draft_update" || isSocialAction(data?.action)) {
        return clearError(current);
      }

      return withError(current, code, {
        connectionStatus:
          current.connectionStatus === "connecting" ? "disconnected" : current.connectionStatus,
        sessionId: isRejectedSessionError(code) ? "" : current.sessionId,
      });
    }
    default:
      return withError(current, "unknown_message_type");
  }
}

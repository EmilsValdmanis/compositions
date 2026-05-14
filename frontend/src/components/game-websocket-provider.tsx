import { createContext, useContext, useEffect, useRef, useState } from "react";
import { getGameConnectionAuth } from "#/lib/game-auth";

type ConnectionStatus = "idle" | "disconnected" | "connecting" | "connected";

type PlayerSnapshot = {
  playerId: string;
  sessionId: string;
  name: string;
  imageUrl?: string;
  connected: boolean;
  seat: number;
  isHost: boolean;
  canReconnect: boolean;
};

type RoomSnapshot = {
  code: string;
  phase: string;
  hostPlayerId: string;
  dealerIndex?: number;
  players: PlayerSnapshot[];
};

type LobbyState = {
  connectionStatus: ConnectionStatus;
  sessionId: string;
  playerId: string;
  room: RoomSnapshot | null;
  lastError: string | null;
  lastErrorId: number;
  lastEvent: string | null;
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
  leaveRoom: () => void;
};

const initialState: LobbyState = {
  connectionStatus: "idle",
  sessionId: "",
  playerId: "",
  room: null,
  lastError: null,
  lastErrorId: 0,
  lastEvent: null,
};

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
      lastEvent: "room_state",
    }),
  left_room: (current) =>
    clearError(current, {
      room: null,
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
  const [state, setState] = useState(initialState);

  useEffect(() => {
    return () => {
      socketRef.current?.close();
    };
  }, []);

  function updateState(updater: (current: LobbyState) => LobbyState) {
    setState((current) => updater(current));
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

    const connectionAuth = await getConnectionAuth();
    if (!connectionAuth?.authToken) {
      setConnectionError("authentication required before connecting");
      return;
    }

    const wsUrl = new URL("/ws", serverUrl).toString();

    const socket = new WebSocket(wsUrl);
    const sessionId = state.sessionId;
    const authToken = connectionAuth.authToken;
    socketRef.current = socket;

    updateState((current) =>
      clearError(current, {
        connectionStatus: "connecting",
        lastEvent: null,
      }),
    );

    socket.onopen = () => {
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
      setConnectionError("failed to connect websocket");
    };

    socket.onclose = () => {
      if (socketRef.current === socket) {
        socketRef.current = null;

        updateState((current) => ({
          ...current,
          connectionStatus: "disconnected",
        }));
      }
    };
  }

  function disconnect() {
    socketRef.current?.close();
    updateState((current) => ({
      ...current,
      room: null,
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
    leaveRoom: () => send("leave_room", {}),
  };

  return <GameWebSocketContext value={value}>{children}</GameWebSocketContext>;
}

export function useGameWebSocket() {
  const context = useContext(GameWebSocketContext);

  if (!context) {
    throw new Error("useGameWebSocket must be used within a GameWebSocketProvider");
  }

  return context;
}

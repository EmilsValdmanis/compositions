import { createContext, useContext, useEffect, useRef, useState } from "react";

type ConnectionStatus = "disconnected" | "connecting" | "connected";

type PlayerSnapshot = {
  playerId: string;
  sessionId: string;
  name: string;
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
  lastEvent: string | null;
};

type Envelope<T = unknown> = {
  type: string;
  data?: T;
};

type GameWebSocketContextValue = {
  state: LobbyState;
  isConnected: boolean;
  connect: () => void;
  disconnect: () => void;
  createRoom: (name: string) => void;
  joinRoom: (roomCode: string, name: string) => void;
  startGame: () => void;
  leaveRoom: () => void;
};

const initialState: LobbyState = {
  connectionStatus: "disconnected",
  sessionId: "",
  playerId: "",
  room: null,
  lastError: null,
  lastEvent: null,
};

const GameWebSocketContext = createContext<GameWebSocketContextValue | null>(null);

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

  function send(type: string, data: unknown) {
    const socket = socketRef.current;

    if (!socket || socket.readyState !== WebSocket.OPEN) {
      updateState((current) => ({
        ...current,
        lastError: "websocket is not connected",
      }));
      return;
    }

    socket.send(JSON.stringify({ type, data }));
    updateState((current) => ({
      ...current,
      lastEvent: type,
      lastError: null,
    }));
  }

  function connect() {
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
      updateState((current) => ({
        ...current,
        lastError: "missing VITE_GAME_SERVER_URL",
      }));
      return;
    }

    const wsUrl = new URL("/ws", serverUrl).toString();
    const socket = new WebSocket(wsUrl);
    const sessionId = state.sessionId;
    socketRef.current = socket;

    updateState((current) => ({
      ...current,
      connectionStatus: "connecting",
      lastError: null,
      lastEvent: null,
    }));

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "connect", data: { sessionId } }));
    };

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as Envelope<any>;

      switch (message.type) {
        case "connected":
          updateState((current) => ({
            ...current,
            connectionStatus: "connected",
            sessionId: message.data?.sessionId ?? current.sessionId,
            playerId: message.data?.playerId ?? current.playerId,
            lastError: null,
            lastEvent: "connected",
          }));
          return;
        case "room_state":
          updateState((current) => ({
            ...current,
            room: message.data?.room ?? null,
            lastError: null,
            lastEvent: "room_state",
          }));
          return;
        case "left_room":
          updateState((current) => ({
            ...current,
            room: null,
            lastError: null,
            lastEvent: "left_room",
          }));
          return;
        case "error":
          updateState((current) => ({
            ...current,
            lastError: message.data?.message ?? "unknown error",
            lastEvent: "error",
          }));
          return;
      }
    };

    socket.onerror = () => {
      updateState((current) => ({
        ...current,
        connectionStatus: "disconnected",
        lastError: "failed to connect websocket",
      }));
    };

    socket.onclose = () => {
      if (socketRef.current === socket) {
        socketRef.current = null;
      }

      updateState((current) => ({
        ...current,
        connectionStatus: "disconnected",
      }));
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
    isConnected: state.connectionStatus === "connected",
    connect,
    disconnect,
    createRoom: (name) => send("create_room", { name }),
    joinRoom: (roomCode, name) => send("join_room", { roomCode, name }),
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

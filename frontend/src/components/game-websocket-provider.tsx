import { createContext, use, useEffect, useState } from "react";
import { useSelector } from "@tanstack/react-store";
import { createGameConnection } from "#/lib/game-connection";

const GameWebSocketContext = createContext<ReturnType<typeof createGameConnection> | null>(null);

export function GameWebSocketProvider({ children }: { children: React.ReactNode }) {
  const [connection] = useState(createGameConnection);

  useEffect(() => {
    window.addEventListener("beforeunload", connection.stop);
    window.addEventListener("pagehide", connection.stop);

    return () => {
      window.removeEventListener("beforeunload", connection.stop);
      window.removeEventListener("pagehide", connection.stop);
      connection.stop();
    };
  }, [connection]);

  return <GameWebSocketContext value={connection}>{children}</GameWebSocketContext>;
}

export function useGameWebSocket() {
  const connection = use(GameWebSocketContext);
  if (!connection) {
    throw new Error("useGameWebSocket must be used within a GameWebSocketProvider");
  }

  const state = useSelector(connection.store);
  return { ...connection.commands, state };
}

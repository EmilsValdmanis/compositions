import { useEffect, useEffectEvent } from "react";
import { Route } from "#/routes/__root";
import { useGameWebSocket } from "#/components/game-websocket-provider";

export function AutoConnectWebSocket() {
  const { connect, disconnect } = useGameWebSocket();
  const { session } = Route.useRouteContext();
  const isAuthenticated = !!session;
  const syncConnection = useEffectEvent(() => {
    if (isAuthenticated) {
      void connect();
      return;
    }

    disconnect();
  });

  useEffect(() => {
    syncConnection();
  }, [isAuthenticated]);

  return null;
}

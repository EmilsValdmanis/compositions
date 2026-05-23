import { useEffect, useEffectEvent } from "react";
import { getRouteApi } from "@tanstack/react-router";
import { useGameWebSocket } from "#/components/game-websocket-provider";

const rootRouteApi = getRouteApi("__root__");

export function AutoConnectWebSocket() {
  const { connect, disconnect } = useGameWebSocket();
  const { session } = rootRouteApi.useRouteContext();
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

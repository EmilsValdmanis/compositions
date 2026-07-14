import { type LobbyState } from "#/components/game-websocket-provider";

type GameRouteState = Pick<LobbyState, "connectionStatus" | "room" | "game">;

export function isGameRouteSnapshotResolving(state: GameRouteState) {
  if (state.connectionStatus === "idle") {
    return true;
  }

  if (state.connectionStatus === "connecting" && state.room === null && state.game === null) {
    return true;
  }

  return state.room !== null && state.room.phase !== "lobby" && state.game === null;
}

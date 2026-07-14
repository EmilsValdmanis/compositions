import { describe, expect, it } from "vite-plus/test";
import { type GameSnapshot, type RoomSnapshot } from "#/components/game-websocket-provider";
import { isGameRouteSnapshotResolving } from "#/components/routes/game-route-view-state";

const room = (phase: string): RoomSnapshot => ({
  code: "ROOM",
  phase,
  hostPlayerId: "player-1",
  players: [],
});

const game = {} as GameSnapshot;

describe("isGameRouteSnapshotResolving", () => {
  it("keeps the loading view while a restored game room is waiting for its game snapshot", () => {
    expect(
      isGameRouteSnapshotResolving({
        connectionStatus: "connected",
        room: room("game_over"),
        game: null,
      }),
    ).toBe(true);
  });

  it("shows the resolved game view once its snapshot arrives", () => {
    expect(
      isGameRouteSnapshotResolving({
        connectionStatus: "connected",
        room: room("game_over"),
        game,
      }),
    ).toBe(false);
  });

  it("does not block a connected lobby without a game snapshot", () => {
    expect(
      isGameRouteSnapshotResolving({
        connectionStatus: "connected",
        room: room("lobby"),
        game: null,
      }),
    ).toBe(false);
  });
});

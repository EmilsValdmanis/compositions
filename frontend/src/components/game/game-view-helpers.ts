import { type PlayerSnapshot, type RoomSnapshot } from "#/components/game-websocket-provider";
import { m } from "#/paraglide/messages.js";

export function playerName(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId)?.name ?? m.waiting();
}

export function playerById(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId) ?? null;
}

export function playersForResults(resultRoom: RoomSnapshot | null, liveRoom: RoomSnapshot | null) {
  if (resultRoom && liveRoom?.code === resultRoom.code) {
    return liveRoom.players;
  }

  return resultRoom?.players ?? [];
}

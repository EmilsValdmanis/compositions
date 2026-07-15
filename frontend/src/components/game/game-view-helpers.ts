import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { m } from "#/paraglide/messages.js";

export function playerName(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId)?.name ?? m.waiting();
}

export function playerById(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId) ?? null;
}

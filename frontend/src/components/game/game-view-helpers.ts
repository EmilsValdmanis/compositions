import { type PlayerSnapshot } from "#/components/game-websocket-provider";

export function compactId(value: string) {
  if (!value) {
    return "None";
  }

  if (value.length <= 12) {
    return value;
  }

  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

export function playerName(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId)?.name ?? "Waiting";
}

export function playerById(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId) ?? null;
}

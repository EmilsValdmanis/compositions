import { type PlayerSnapshot } from "#/components/game-websocket-provider";

const gamePhaseLabels: Record<number, string> = {
  0: "lobby",
  1: "in_progress",
  2: "round_over",
  3: "game_over",
};

export function formatLabel(value: string | number | null | undefined) {
  const label =
    typeof value === "number" ? (gamePhaseLabels[value] ?? String(value)) : (value ?? "");

  return label
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

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

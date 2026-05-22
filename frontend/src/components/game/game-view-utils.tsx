import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { getUserInitials } from "#/lib/utils";

const gamePhaseLabels: Record<number, string> = {
  0: "lobby",
  1: "in_progress",
  2: "round_over",
  3: "game_over",
};

export function formatLabel(value: string | number | null | undefined) {
  const label =
    typeof value === "number" ? (gamePhaseLabels[value] ?? String(value)) : (value ?? "");

  const words: string[] = [];

  for (const part of label.split("_")) {
    if (!part) {
      continue;
    }

    words.push(part.charAt(0).toUpperCase() + part.slice(1));
  }

  return words.join(" ");
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

export function playerById(players: PlayerSnapshot[], playerId?: string) {
  return players.find((player) => player.playerId === playerId) ?? null;
}

export function PlayerMarker({
  players,
  playerId,
  className,
}: {
  players: PlayerSnapshot[];
  playerId?: string;
  className?: string;
}) {
  const player = playerById(players, playerId);

  if (!player) {
    return null;
  }

  return (
    <Avatar size="sm" className={className}>
      {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
      <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
    </Avatar>
  );
}

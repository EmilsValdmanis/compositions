import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { playerById } from "#/components/game/game-view-helpers";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { cn, getUserInitials } from "#/lib/utils";

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

export function ActivityLabel({
  players,
  playerId,
  label = "New",
  className,
}: {
  players: PlayerSnapshot[];
  playerId?: string;
  label?: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-1 text-[0.65rem] font-medium text-muted-foreground",
        className,
      )}
    >
      <span className="uppercase tracking-wide">{label}</span>
      {playerId ? <PlayerMarker players={players} playerId={playerId} className="size-4" /> : null}
    </div>
  );
}

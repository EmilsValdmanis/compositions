import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { playerById } from "#/components/game/game-view-helpers";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { getUserInitials } from "#/lib/utils";

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

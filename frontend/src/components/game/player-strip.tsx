import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { cn, getUserInitials } from "#/lib/utils";

export function PlayerStrip({
  players,
  game,
  showHostBadges = true,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  showHostBadges?: boolean;
}) {
  const playerStates = game?.players ?? [];
  return (
    <div className="grid gap-2">
      {players.map((player) => {
        const gamePlayer = playerStates.find((item) => item.playerId === player.playerId);
        const isTurn = game?.turn.playerId === player.playerId;
        return (
          <div
            key={player.playerId}
            className={cn(
              "flex items-center justify-between gap-3 rounded-3xl border px-3 py-2",
              isTurn ? "border-primary/40 bg-primary/10" : "border-border/60 bg-muted/20",
            )}
          >
            <div className="flex min-w-0 items-center gap-3">
              <Avatar>
                {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
                <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
                <AvatarBadge
                  className={cn("ring-border", player.connected ? "bg-primary" : "bg-destructive")}
                />
              </Avatar>
              <div className="min-w-0">
                <p className="truncate font-medium">{player.name}</p>
              </div>
            </div>
            <div className="flex flex-wrap justify-end gap-1.5">
              {showHostBadges && player.isHost ? <Badge variant="secondary">Host</Badge> : null}
              {gamePlayer ? <Badge variant="outline">{gamePlayer.handCount} cards</Badge> : null}
            </div>
          </div>
        );
      })}
    </div>
  );
}

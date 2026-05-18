import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { getUserInitials } from "#/lib/utils";

export function PlayerStrip({
  players,
  game,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
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
            className={`flex items-center justify-between gap-3 rounded-3xl border px-3 py-2 ${
              isTurn ? "border-primary/40 bg-primary/10" : "border-border/60 bg-muted/20"
            }`}
          >
            <div className="flex min-w-0 items-center gap-3">
              <Avatar>
                {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
                <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
              </Avatar>
              <div className="min-w-0">
                <p className="truncate font-medium">{player.name}</p>
                <p className="text-xs text-muted-foreground">Seat #{player.seat + 1}</p>
              </div>
            </div>
            <div className="flex flex-wrap justify-end gap-1.5">
              {isTurn ? <Badge>Turn</Badge> : null}
              {player.isHost ? <Badge variant="secondary">Host</Badge> : null}
              {gamePlayer ? <Badge variant="outline">{gamePlayer.handCount} cards</Badge> : null}
              <Badge variant={player.connected ? "default" : "outline"}>
                {player.connected ? "Online" : "Offline"}
              </Badge>
            </div>
          </div>
        );
      })}
    </div>
  );
}

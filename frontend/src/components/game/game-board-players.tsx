import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import { Badge } from "../ui/badge";

export function GameBoardPlayers({
  players,
  game,
  connectedPlayers,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  connectedPlayers: number;
}) {
  return (
    <Card size="sm" className="overflow-y-scroll grow">
      <CardHeader>
        <CardTitle>Players</CardTitle>
        <CardAction>
          <Badge variant="outline">
            {connectedPlayers}/{players.length || 0} online
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <PlayerStrip players={players} game={game} />
      </CardContent>
    </Card>
  );
}

import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";

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
        <CardDescription>
          {connectedPlayers}/{players.length || 0} online
        </CardDescription>
      </CardHeader>
      <CardContent>
        <PlayerStrip players={players} game={game} />
      </CardContent>
    </Card>
  );
}

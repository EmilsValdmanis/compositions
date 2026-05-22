import {
  type CompletedGameSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";

type GameResultsViewProps = {
  room: CompletedGameSnapshot["room"] | null;
  game: GameSnapshot;
  players: PlayerSnapshot[];
  playerId: string;
  onStartNextRound?: () => void;
};

function rankingRows(game: GameSnapshot, players: PlayerSnapshot[]) {
  return [...game.players]
    .sort((left, right) => left.totalPoints - right.totalPoints)
    .map((playerState, index) => ({
      rank: index + 1,
      player: players.find((player) => player.playerId === playerState.playerId) ?? null,
      playerState,
    }));
}

export function GameResultsView({
  room,
  game,
  players,
  playerId,
  onStartNextRound,
}: GameResultsViewProps) {
  const winner = players.find(
    (player) => player.playerId === game.players[game.roundWinnerIndex]?.playerId,
  );
  const isGameOver = room?.phase === "game_over";
  const isHost = room?.hostPlayerId === playerId;

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col justify-center gap-4">
      <Card>
        <CardHeader>
          <CardTitle>{isGameOver ? "Game finished" : `Round ${game.round} complete`}</CardTitle>
          <CardDescription>
            {isGameOver
              ? `${winner?.name ?? "A player"} wins the game.`
              : `${winner?.name ?? "A player"} won the round.`}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="flex flex-wrap items-center gap-2">
            <Badge>{winner?.name ?? "Unknown winner"}</Badge>
            <Badge variant="outline">Round {game.round}</Badge>
            <Badge variant="outline">Dealer seat #{game.dealerIndex + 1}</Badge>
            {isGameOver ? <Badge variant="secondary">Victor</Badge> : null}
          </div>

          <div className="grid gap-2">
            {rankingRows(game, players).map(({ rank, player, playerState }) => (
              <div
                key={playerState.playerId}
                className="flex items-center justify-between gap-3 rounded-3xl border border-border/70 bg-muted/20 px-4 py-3"
              >
                <div>
                  <p className="font-medium">
                    #{rank} {player?.name ?? "Unknown player"}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {playerState.handCount} cards left, {playerState.totalPoints} total points
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  {playerState.playerId === game.players[game.roundWinnerIndex]?.playerId ? (
                    <Badge>Winner</Badge>
                  ) : null}
                  {playerState.playerId === playerId ? <Badge variant="outline">You</Badge> : null}
                </div>
              </div>
            ))}
          </div>

          {!isGameOver ? (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-3xl border border-border/70 bg-muted/20 p-4">
              <p className="text-sm text-muted-foreground">
                {isHost
                  ? "Start the next round when everyone is ready."
                  : "Waiting for the lobby owner to start the next round."}
              </p>
              <Button type="button" onClick={onStartNextRound} disabled={!isHost}>
                Start next round
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}

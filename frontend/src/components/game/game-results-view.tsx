import {
  type CompletedGameSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Separator } from "#/components/ui/separator";

type GameResultsViewProps = {
  room: CompletedGameSnapshot["room"] | null;
  game: GameSnapshot;
  players: PlayerSnapshot[];
  playerId: string;
  onStartNextRound?: () => void;
};

function rankingRows(game: GameSnapshot, players: PlayerSnapshot[]) {
  return game.players
    .toSorted((left, right) => left.totalPoints - right.totalPoints)
    .map((playerState, index) => ({
      rank: index + 1,
      player: players.find((player) => player.playerId === playerState.playerId) ?? null,
      playerState,
    }));
}

function pointsLabel(points: number) {
  if (points <= 0) {
    return "No points added";
  }

  return `+${points} pts`;
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
          <div className="grid gap-3 rounded-3xl border border-border/70 bg-muted/20 p-4 sm:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] sm:items-end">
            <div className="grid gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <Badge>{winner?.name ?? "Unknown winner"}</Badge>
                <Badge variant="outline">Round {game.round}</Badge>
                <Badge variant="outline">Dealer seat #{game.dealerIndex + 1}</Badge>
                {isGameOver ? <Badge variant="secondary">Victor</Badge> : null}
              </div>
              <p className="text-sm text-muted-foreground">
                Round scoring is shown per player so everyone can scan how the standings changed.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 rounded-3xl border border-border/70 bg-background/80 p-3 text-sm">
              <div>
                <p className="text-muted-foreground">Players</p>
                <p className="font-medium">{game.players.length}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Points added</p>
                <p className="font-medium">
                  {game.players.reduce((sum, player) => sum + player.pointsGained, 0)} pts
                </p>
              </div>
            </div>
          </div>

          <div className="grid gap-2">
            {rankingRows(game, players).map(({ rank, player, playerState }) => (
              <div
                key={playerState.playerId}
                className="grid gap-3 rounded-3xl border border-border/70 bg-muted/20 px-4 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
              >
                <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-medium">
                        #{rank} {player?.name ?? "Unknown player"}
                      </p>
                      {playerState.playerId === game.players[game.roundWinnerIndex]?.playerId ? (
                        <Badge>Winner</Badge>
                      ) : null}
                      {playerState.playerId === playerId ? (
                        <Badge variant="outline">You</Badge>
                      ) : null}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {playerState.handCount} cards left after the round
                    </p>
                  </div>

                  <div className="min-w-40 rounded-2xl border border-border/70 bg-background/80 px-3 py-2 text-right">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      Round change
                    </p>
                    <p className="text-lg font-semibold tracking-tight">
                      {pointsLabel(playerState.pointsGained)}
                    </p>
                  </div>
                </div>

                <div className="grid gap-2 text-sm sm:min-w-44 sm:text-right">
                  <div>
                    <p className="text-muted-foreground">Total score</p>
                    <p className="font-medium">{playerState.totalPoints} pts</p>
                  </div>
                  <Separator className="sm:hidden" />
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

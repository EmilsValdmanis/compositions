import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";

type DealChoiceState = {
  pendingDealChoice: PendingDealChoiceSnapshot | null;
  dealChooserName: string | null;
  isDealChooser: boolean;
};

type GameResultsViewProps = {
  room: CompletedGameSnapshot["room"] | null;
  game: GameSnapshot;
  players: PlayerSnapshot[];
  playerId: string;
  dealChoice: DealChoiceState;
  onStartNextRound?: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
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

export function GameResultsView({
  room,
  game,
  players,
  playerId,
  dealChoice,
  onStartNextRound,
  onChooseDealing,
}: GameResultsViewProps) {
  const winner = players.find(
    (player) => player.playerId === game.players[game.roundWinnerIndex]?.playerId,
  );
  const isGameOver = room?.phase === "game_over";
  const isHost = room?.hostPlayerId === playerId;

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col justify-center gap-4">
      <Card className="border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>{isGameOver ? "Game finished" : "Round complete"}</CardTitle>
              <CardDescription>
                {winner?.name ?? "A player"} {isGameOver ? "wins the game" : "won the round"}
              </CardDescription>
            </div>
            <Badge variant={isGameOver ? "default" : "secondary"}>
              {isGameOver ? "Final" : "Complete"}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="overflow-hidden rounded-lg border border-border/70">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-14">#</TableHead>
                  <TableHead>Player</TableHead>
                  <TableHead className="text-right">Cards</TableHead>
                  <TableHead className="text-right">Score</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rankingRows(game, players).map(({ rank, player, playerState }) => (
                  <TableRow key={playerState.playerId}>
                    <TableCell className="font-medium">{rank}</TableCell>
                    <TableCell>
                      <div className="flex min-w-40 flex-wrap items-center gap-2">
                        <span className="font-medium">{player?.name ?? "Unknown player"}</span>
                        {playerState.playerId === game.players[game.roundWinnerIndex]?.playerId ? (
                          <Badge>Winner</Badge>
                        ) : null}
                        {playerState.playerId === playerId ? (
                          <Badge variant="outline">You</Badge>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{playerState.handCount}</TableCell>
                    <TableCell className="text-right font-medium">
                      {playerState.totalPoints}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {!isGameOver ? (
            dealChoice.pendingDealChoice ? (
              <DealChoicePanel
                players={players}
                pendingDealChoice={dealChoice.pendingDealChoice}
                dealChooserName={dealChoice.dealChooserName}
                isDealChooser={dealChoice.isDealChooser}
                onChooseDealing={onChooseDealing}
              />
            ) : (
              <div className="flex flex-wrap items-center justify-end gap-3">
                {!isHost ? (
                  <p className="text-sm text-muted-foreground">Waiting for the host.</p>
                ) : null}
                <Button type="button" onClick={onStartNextRound} disabled={!isHost}>
                  Start next round
                </Button>
              </div>
            )
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}

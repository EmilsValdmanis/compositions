import { Cards01Icon, UserIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { GameCard } from "#/components/game/game-card";
import { cardName } from "#/components/game/game-card-utils";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
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
import { Tooltip, TooltipContent, TooltipTrigger } from "#/components/ui/tooltip";
import { cn, getUserInitials } from "#/lib/utils";

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
  connectedPlayers: number;
  dealChoice: DealChoiceState;
  onStartNextRound?: () => void;
  onReturnToLobby?: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
  onSendEmote: (emoji: string) => void;
};

function rankingRows(game: GameSnapshot, players: PlayerSnapshot[]) {
  const roundWinnerPlayerId = game.players[game.roundWinnerIndex]?.playerId;

  return game.players
    .toSorted((left, right) => left.totalPoints - right.totalPoints)
    .map((playerState, index) => ({
      rank: index + 1,
      player: players.find((player) => player.playerId === playerState.playerId) ?? null,
      playerState,
      isRoundWinner: playerState.playerId === roundWinnerPlayerId,
    }));
}

function pointsGainedLabel(pointsGained: number) {
  return pointsGained > 0 ? `+${pointsGained}` : "0";
}

function noopResetDraftCompositions() {}

function LeftoverHandTooltip({
  handCount,
  hand,
  playerName,
}: {
  handCount: number;
  hand: GameSnapshot["hand"];
  playerName: string;
}) {
  const hasRevealedHand = hand && hand.length > 0;
  const handTitle = hasRevealedHand
    ? hand.map((card) => cardName(card)).join(", ")
    : `${handCount} cards left`;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="ml-auto w-fit tabular-nums"
            disabled={!hasRevealedHand}
            title={handTitle}
            aria-label={
              hasRevealedHand
                ? `${playerName}'s leftover hand`
                : `${playerName} has ${handCount} cards left`
            }
          />
        }
      >
        <span>{handCount}</span>
        <HugeiconsIcon icon={Cards01Icon} data-icon="inline-end" />
      </TooltipTrigger>
      <TooltipContent className="flex flex-wrap gap-1 py-2.5 max-w-60">
        {hand.map((card, index) => (
          <GameCard key={`${cardName(card)}-${index}`} card={card} size="compact" />
        ))}
      </TooltipContent>
    </Tooltip>
  );
}

export function GameResultsView({
  room,
  game,
  players,
  playerId,
  connectedPlayers,
  dealChoice,
  onStartNextRound,
  onReturnToLobby,
  onChooseDealing,
  onSendEmote,
}: GameResultsViewProps) {
  const winner = players.find(
    (player) => player.playerId === game.players[game.roundWinnerIndex]?.playerId,
  );
  const isGameOver = room?.phase === "game_over";
  const isHost = room?.hostPlayerId === playerId;

  return (
    <div className="mx-auto grid w-full max-w-5xl flex-1 content-center gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
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
                  <TableHead className="w-16 text-right">Round</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rankingRows(game, players).map(({ rank, player, playerState, isRoundWinner }) => {
                  const playerName = player?.name ?? "Unknown player";

                  return (
                    <TableRow
                      key={playerState.playerId}
                      className={cn(
                        isRoundWinner && "border-primary/35 bg-primary/10 hover:bg-primary/15",
                      )}
                    >
                      <TableCell className={cn("font-medium", isRoundWinner && "text-primary")}>
                        {rank}
                      </TableCell>
                      <TableCell>
                        <div className="flex min-w-40 flex-wrap items-center gap-2">
                          <Avatar size="sm">
                            {player?.imageUrl ? (
                              <AvatarImage src={player.imageUrl} alt={playerName} />
                            ) : null}
                            <AvatarFallback>
                              {playerName === "Unknown player" ? (
                                <HugeiconsIcon icon={UserIcon} strokeWidth={2} />
                              ) : (
                                getUserInitials(playerName)
                              )}
                            </AvatarFallback>
                          </Avatar>
                          <span className={cn("font-medium", isRoundWinner && "text-primary")}>
                            {playerName}
                          </span>
                          {isRoundWinner ? <Badge>Winner</Badge> : null}
                          {playerState.playerId === playerId ? (
                            <Badge variant="outline">You</Badge>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <LeftoverHandTooltip
                          handCount={playerState.handCount}
                          hand={playerState.hand ?? []}
                          playerName={playerName}
                        />
                      </TableCell>
                      <TableCell className="text-right font-medium tabular-nums">
                        {playerState.totalPoints}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {playerState.pointsGained > 0 ? (
                          <span className="text-xs font-medium text-primary">
                            {pointsGainedLabel(playerState.pointsGained)}
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">0</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>

          {isGameOver ? (
            <div className="flex flex-wrap items-center justify-end gap-3">
              <Button type="button" onClick={onReturnToLobby}>
                Back to lobby
              </Button>
            </div>
          ) : dealChoice.pendingDealChoice ? (
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
          )}
        </CardContent>
      </Card>

      <GameBoardPlayers
        players={players}
        game={game}
        connectedPlayers={connectedPlayers}
        hasDraftedCompositions={false}
        showTurnIndicator={false}
        onResetDraftCompositions={noopResetDraftCompositions}
        onSendEmote={onSendEmote}
      />
    </div>
  );
}

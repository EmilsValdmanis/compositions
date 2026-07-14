import { useEffect, useRef, useState } from "react";
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
import { AnimatedNumber } from "#/components/ui/animated-number";
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
import { P, Strong, Text } from "#/components/typography";
import { fireCelebrationConfetti } from "#/lib/confetti";
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

function noopResetDraftCompositions() {}

function ResultPoints({
  totalPoints,
  pointsGained,
}: {
  totalPoints: number;
  pointsGained: number;
}) {
  const [displayedPoints, setDisplayedPoints] = useState(() => ({
    total: Math.max(0, totalPoints - pointsGained),
    gained: 0,
  }));

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      setDisplayedPoints({ total: totalPoints, gained: pointsGained });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [pointsGained, totalPoints]);

  return (
    <>
      <TableCell className="text-right">
        <Text variant="numeric">
          <AnimatedNumber value={displayedPoints.total} />
        </Text>
      </TableCell>
      <TableCell className="text-right">
        <Text
          variant={displayedPoints.gained > 0 ? "caption-strong" : "caption"}
          className={cn(displayedPoints.gained > 0 ? "text-primary" : "text-muted-foreground")}
        >
          <AnimatedNumber
            value={displayedPoints.gained}
            prefix={displayedPoints.gained > 0 ? "+" : undefined}
          />
        </Text>
      </TableCell>
    </>
  );
}

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
            className="ml-auto w-fit"
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
        <Text variant="numeric">
          <AnimatedNumber value={handCount} />
        </Text>
        <HugeiconsIcon icon={Cards01Icon} data-icon="inline-end" />
      </TooltipTrigger>
      <TooltipContent className="flex flex-wrap justify-center gap-1 py-2.5 max-w-53">
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
    (player) =>
      player.playerId ===
      (room?.conclusion?.winnerPlayerId ?? game.players[game.roundWinnerIndex]?.playerId),
  );
  const isGameOver = room?.phase === "game_over";
  const isHost = room?.hostPlayerId === playerId;
  const conclusion = room?.conclusion;
  const resultTitle =
    conclusion?.kind === "technical_abort"
      ? "Game aborted"
      : conclusion?.kind === "mutual_end"
        ? "Game ended"
        : isGameOver
          ? "Game finished"
          : "Round complete";
  const resultDescription =
    conclusion?.kind === "technical_abort"
      ? "Ended after a reported game problem"
      : conclusion?.kind === "mutual_end"
        ? "Every active player agreed to end without a winner"
        : conclusion?.kind === "forfeit"
          ? `${winner?.name ?? "A player"} wins by forfeit`
          : `${winner?.name ?? "A player"} ${isGameOver ? "wins the game" : "won the round"}`;
  const resultBadge =
    conclusion?.kind === "technical_abort"
      ? "Aborted"
      : conclusion?.kind === "mutual_end"
        ? "No winner"
        : isGameOver
          ? "Final"
          : "Complete";
  const hasCelebrated = useRef(false);
  const isWinner = winner?.playerId === playerId;

  useEffect(() => {
    if (!isWinner || hasCelebrated.current) return;

    hasCelebrated.current = true;
    void fireCelebrationConfetti({ delayMs: 250 });
  }, [isWinner]);

  return (
    <div className="mx-auto grid w-full max-w-5xl flex-1 content-center gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <Card className="border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>{resultTitle}</CardTitle>
              <CardDescription>{resultDescription}</CardDescription>
            </div>
            <Badge variant={isGameOver ? "default" : "secondary"}>{resultBadge}</Badge>
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
                      <TableCell>
                        <Strong className={cn(isRoundWinner && "text-primary")}>{rank}</Strong>
                      </TableCell>
                      <TableCell>
                        <div className="flex min-w-40 flex-wrap items-center gap-2">
                          <Avatar
                            size="sm"
                            aria-label={isRoundWinner ? `${playerName}, winner` : playerName}
                          >
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
                            {isRoundWinner ? (
                              <Text
                                variant="emoji-sm"
                                aria-hidden="true"
                                className="absolute -top-3 -right-0.5 z-20 origin-bottom-left rotate-18 drop-shadow-sm"
                                style={{
                                  fontFamily:
                                    '"Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji"',
                                }}
                              >
                                👑
                              </Text>
                            ) : null}
                          </Avatar>
                          <Strong className={cn(isRoundWinner && "text-primary")}>
                            {playerName}
                          </Strong>
                          {playerState.playerId === playerId ? (
                            <Badge variant="outline">You</Badge>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        {playerState.hand && playerState.hand?.length !== 0 && (
                          <LeftoverHandTooltip
                            handCount={playerState.handCount}
                            hand={playerState.hand}
                            playerName={playerName}
                          />
                        )}
                      </TableCell>
                      <ResultPoints
                        totalPoints={playerState.totalPoints}
                        pointsGained={playerState.pointsGained}
                      />
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
                <P size="sm" className="text-muted-foreground">
                  Waiting for the host.
                </P>
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

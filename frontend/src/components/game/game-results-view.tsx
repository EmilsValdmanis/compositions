import { useEffect, useRef, useState } from "react";
import { Cards01Icon, Home01Icon, Sent02Icon, UserIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type SocialState,
} from "#/components/game-websocket-provider";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { resultScoreState, type ResultScorePhase } from "#/components/game/game-results-state";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { GameCard } from "#/components/game/game-card";
import { cardName } from "#/components/game/game-card-utils";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "#/components/ui/tooltip";
import { Caption, P } from "#/components/typography";
import { fireCelebrationConfetti } from "#/lib/confetti";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

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
  onBackToLobby?: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
  onSendEmote: (emoji: string) => void;
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
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

function keyedCards(cards: GameSnapshot["hand"]) {
  const occurrences = new Map<string, number>();

  return cards.map((card) => {
    const identity = card.isJoker ? "joker" : `${card.rank ?? "unknown"}-${card.suit ?? "unknown"}`;
    const occurrence = occurrences.get(identity) ?? 0;
    occurrences.set(identity, occurrence + 1);

    return { card, key: `${identity}-${occurrence}` };
  });
}

function ResultPoints({
  playerState,
  phase,
}: {
  playerState: GameSnapshot["players"][number];
  phase: ResultScorePhase;
}) {
  const score = resultScoreState(playerState, phase);

  return (
    <>
      <TableCell
        className="text-right"
        data-score-phase={phase}
        data-flying={score.hasAdjustment || undefined}
      >
        <div className="flex min-h-9 items-center justify-end">
          <P
            size="sm"
            data-flying-value={score.isShowingAdjustment || undefined}
            className={cn(
              "relative font-medium tabular-nums transition-colors duration-200",
              score.isShowingOverHundred && "text-destructive",
              score.isShowingAdjustment && "text-primary",
            )}
            title={
              score.isShowingAdjustment
                ? m.flying_score_adjustment({
                    from: score.unadjustedTotal,
                    to: playerState.totalPoints,
                  })
                : undefined
            }
          >
            {score.isShowingAdjustment ? (
              <>
                <span
                  className="absolute top-1/2 right-[calc(100%+0.75rem)] -translate-y-1/2"
                  aria-hidden="true"
                >
                  <span data-flying-icon className="flying-score-plane block size-4 text-primary">
                    <HugeiconsIcon icon={Sent02Icon} className="size-4" strokeWidth={2} />
                  </span>
                </span>
                <span
                  data-flying-ring
                  className="pointer-events-none absolute top-1/2 left-1/2 size-8 -translate-x-1/2 -translate-y-1/2 rounded-full border border-primary"
                  aria-hidden="true"
                />
              </>
            ) : null}
            <AnimatedNumber value={score.displayedTotal} />
          </P>
        </div>
      </TableCell>
      <TableCell className="text-right">
        <Caption className={cn(score.displayedGained > 0 && "font-medium text-primary")}>
          <AnimatedNumber
            value={score.displayedGained}
            prefix={score.displayedGained > 0 ? "+" : undefined}
          />
        </Caption>
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
    : m.cards_left({ count: handCount });

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
                ? m.leftover_hand({ name: playerName })
                : m.player_cards_left({ name: playerName, count: handCount })
            }
          />
        }
      >
        <P size="sm" className="font-medium tabular-nums">
          <AnimatedNumber value={handCount} />
        </P>
        <HugeiconsIcon icon={Cards01Icon} data-icon="inline-end" />
      </TooltipTrigger>
      <TooltipContent className="flex flex-wrap justify-center gap-1 py-2.5 max-w-53">
        {keyedCards(hand).map(({ card, key }) => (
          <GameCard key={key} card={card} size="compact" />
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
  onBackToLobby,
  onChooseDealing,
  onSendEmote,
  social,
  onSendFriendRequest,
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
      ? m.game_aborted()
      : conclusion?.kind === "mutual_end"
        ? m.game_ended()
        : isGameOver
          ? m.game_finished()
          : m.round_complete();
  const resultDescription =
    conclusion?.kind === "technical_abort"
      ? m.ended_reported_problem()
      : conclusion?.kind === "mutual_end"
        ? m.ended_by_agreement_description()
        : conclusion?.kind === "forfeit"
          ? m.wins_by_forfeit({ name: winner?.name ?? m.a_player() })
          : isGameOver
            ? m.wins_game({ name: winner?.name ?? m.a_player() })
            : m.won_round({ name: winner?.name ?? m.a_player() });
  const resultBadge =
    conclusion?.kind === "technical_abort"
      ? m.aborted()
      : conclusion?.kind === "mutual_end"
        ? m.no_winner()
        : isGameOver
          ? m.final()
          : m.complete();
  const hasCelebrated = useRef(false);
  const isWinner = winner?.playerId === playerId;
  const shouldReduceMotion = useShouldReduceMotion();
  const scoreRevealKey = `${game.round}:${shouldReduceMotion}`;
  const [scoreReveal, setScoreReveal] = useState<{
    key: string;
    phase: ResultScorePhase;
  }>(() => ({ key: scoreRevealKey, phase: "previous" }));
  const scorePhase = shouldReduceMotion
    ? "adjusted"
    : scoreReveal.key === scoreRevealKey
      ? scoreReveal.phase
      : "previous";

  useEffect(() => {
    if (shouldReduceMotion) {
      return;
    }

    const roundScoreTimer = window.setTimeout(
      () => setScoreReveal({ key: scoreRevealKey, phase: "round" }),
      500,
    );
    const adjustmentTimer = window.setTimeout(
      () => setScoreReveal({ key: scoreRevealKey, phase: "adjusted" }),
      1_500,
    );

    return () => {
      window.clearTimeout(roundScoreTimer);
      window.clearTimeout(adjustmentTimer);
    };
  }, [scoreRevealKey, shouldReduceMotion]);

  useEffect(() => {
    if (!isWinner || hasCelebrated.current) return;

    hasCelebrated.current = true;
    void fireCelebrationConfetti({ delayMs: 250 });
  }, [isWinner]);

  return (
    <div className="mx-auto grid w-full max-w-5xl flex-1 content-center gap-4 p-1 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <Card className="min-h-0 border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>{resultTitle}</CardTitle>
              <CardDescription>{resultDescription}</CardDescription>
            </div>
            <Badge variant={isGameOver ? "default" : "secondary"}>{resultBadge}</Badge>
          </div>
        </CardHeader>
        <CardContent className="grid min-h-0 flex-1 auto-rows-max content-start gap-4 overflow-y-auto scroll-fade-x overscroll-y-contain">
          <div className="overflow-hidden rounded-lg border border-border/70">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>#</TableHead>
                  <TableHead>{m.player()}</TableHead>
                  <TableHead className="text-right">{m.cards()}</TableHead>
                  <TableHead className="text-right">{m.score()}</TableHead>
                  <TableHead className="text-right">{m.round()}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rankingRows(game, players).map(({ rank, player, playerState, isRoundWinner }) => {
                  const playerName = player?.name ?? m.unknown_player();

                  return (
                    <TableRow
                      key={playerState.playerId}
                      className={cn(
                        isRoundWinner && "border-primary/35 bg-primary/10 hover:bg-primary/15",
                      )}
                    >
                      <TableCell>
                        <P size="sm" className={cn("font-medium", isRoundWinner && "text-primary")}>
                          {rank}
                        </P>
                      </TableCell>
                      <TableCell>
                        <div className="flex w-fit items-center gap-2">
                          <Avatar
                            size="sm"
                            aria-label={
                              isRoundWinner ? m.winner_accessible({ name: playerName }) : playerName
                            }
                          >
                            {player?.imageUrl ? (
                              <AvatarImage src={player.imageUrl} alt={playerName} />
                            ) : null}
                            <AvatarFallback>
                              {playerName === m.unknown_player() ? (
                                <HugeiconsIcon icon={UserIcon} strokeWidth={2} />
                              ) : (
                                getUserInitials(playerName)
                              )}
                            </AvatarFallback>
                            {isRoundWinner ? (
                              <span
                                aria-hidden="true"
                                className="absolute -top-3 -right-0.5 z-20 origin-bottom-left rotate-18 text-sm/none drop-shadow-sm"
                                style={{
                                  fontFamily:
                                    '"Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji"',
                                }}
                              >
                                👑
                              </span>
                            ) : null}
                          </Avatar>
                          <P
                            size="sm"
                            className={cn("font-medium", isRoundWinner && "text-primary")}
                          >
                            {playerName}
                          </P>
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
                      <ResultPoints playerState={playerState} phase={scorePhase} />
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>

          {!isGameOver && dealChoice.pendingDealChoice ? (
            <DealChoicePanel
              players={players}
              pendingDealChoice={dealChoice.pendingDealChoice}
              dealChooserName={dealChoice.dealChooserName}
              isDealChooser={dealChoice.isDealChooser}
              onChooseDealing={onChooseDealing}
            />
          ) : null}
        </CardContent>
        {isGameOver ? (
          <CardFooter className="shrink-0 justify-end">
            <Button type="button" onClick={onBackToLobby}>
              <HugeiconsIcon icon={Home01Icon} data-icon="inline-start" />
              {m.back_to_lobby()}
            </Button>
          </CardFooter>
        ) : !dealChoice.pendingDealChoice ? (
          <CardFooter className="shrink-0 flex-wrap justify-end gap-3">
            {!isHost ? (
              <P size="sm" className="text-muted-foreground">
                {m.waiting_for_host()}
              </P>
            ) : null}
            <Button type="button" onClick={onStartNextRound} disabled={!isHost}>
              {m.start_next_round()}
            </Button>
          </CardFooter>
        ) : null}
      </Card>

      <GameBoardPlayers
        players={players}
        game={game}
        connectedPlayers={connectedPlayers}
        hasDraftedCompositions={false}
        showTurnIndicator={false}
        onResetDraftCompositions={noopResetDraftCompositions}
        onSendEmote={onSendEmote}
        currentPlayerId={playerId}
        social={social}
        onSendFriendRequest={onSendFriendRequest}
      />
    </div>
  );
}

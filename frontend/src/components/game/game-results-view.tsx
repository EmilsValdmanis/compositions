import { useEffect, useEffectEvent, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
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
import { fireCelebrationConfetti, fireStreamingCelebrationConfetti } from "#/lib/confetti";
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

type ResultScoreTimelinePhase = ResultScorePhase | "counting" | "adjusting";

const SCORE_REVEAL_DELAY_MS = 700;
const SCORE_REVEAL_STEP_MS = 1_000;
const SCORE_COUNT_DURATION_MS = 620;
const SCORE_ADJUSTMENT_HOLD_MS = 180;
const SCORE_ADJUSTMENT_COUNT_DURATION_MS = 520;
const SCORE_REVEAL_HANDOFF_MS = 100;
const RANKING_REORDER_DELAY_MS = 500;
const GAME_WINNER_DURATION_MS = 4_600;

function rankingRows(
  game: GameSnapshot,
  players: PlayerSnapshot[],
  phase: ResultScorePhase = "adjusted",
) {
  const roundWinnerPlayerId = game.players[game.roundWinnerIndex]?.playerId;

  return game.players
    .toSorted((left, right) => {
      const pointsDifference =
        resultScoreState(left, phase).displayedTotal -
        resultScoreState(right, phase).displayedTotal;

      if (pointsDifference !== 0) return pointsDifference;
      return (
        (players.find((player) => player.playerId === left.playerId)?.seat ?? 0) -
        (players.find((player) => player.playerId === right.playerId)?.seat ?? 0)
      );
    })
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

function GradualScoreNumber({
  from,
  value,
  isCounting,
  prefix,
  duration = SCORE_COUNT_DURATION_MS,
  trend = 1,
}: {
  from: number;
  value: number;
  isCounting: boolean;
  prefix?: string;
  duration?: number;
  trend?: 1 | -1;
}) {
  const shouldReduceMotion = useShouldReduceMotion();
  const [displayedValue, setDisplayedValue] = useState(isCounting ? from : value);

  useEffect(() => {
    if (!isCounting || shouldReduceMotion || from === value) return;

    const direction = value > from ? 1 : -1;
    const distance = Math.abs(value - from);
    let currentValue = from;
    const timer = window.setInterval(() => {
      const nextValue = currentValue + direction;
      const hasReachedTarget =
        (direction > 0 && nextValue >= value) || (direction < 0 && nextValue <= value);

      currentValue = hasReachedTarget ? value : nextValue;
      setDisplayedValue(currentValue);

      if (hasReachedTarget) window.clearInterval(timer);
    }, duration / distance);

    return () => window.clearInterval(timer);
  }, [duration, from, isCounting, shouldReduceMotion, value]);

  return (
    <AnimatedNumber
      value={isCounting && !shouldReduceMotion ? displayedValue : value}
      prefix={prefix}
      trend={trend}
    />
  );
}

function ResultPoints({
  playerState,
  phase,
}: {
  playerState: GameSnapshot["players"][number];
  phase: ResultScoreTimelinePhase;
}) {
  const isCountingUp = phase === "counting";
  const isCountingDown = phase === "adjusting";
  const score = resultScoreState(
    playerState,
    isCountingUp ? "round" : isCountingDown ? "adjusted" : phase,
  );
  const previousScore = resultScoreState(playerState, "previous");
  const roundScore = resultScoreState(playerState, "round");
  const showFlyingIndicator = phase === "adjusted" && score.isShowingAdjustment;

  return (
    <>
      <TableCell
        className="relative text-right"
        data-score-phase={phase}
        data-flying={score.hasAdjustment || undefined}
      >
        <div className="flex min-h-9 items-center justify-end">
          <P
            size="sm"
            data-score-value-slot
            data-flying-value={score.isShowingAdjustment || undefined}
            className={cn(
              "relative inline-block w-[3ch] text-right font-medium tabular-nums transition-colors duration-200",
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
            <span data-score-total className="relative inline-block">
              {showFlyingIndicator ? (
                <span
                  className="absolute top-1/2 right-[calc(100%+0.25rem)] -translate-y-1/2"
                  aria-hidden="true"
                >
                  <span data-flying-icon className="flying-score-plane block size-4 text-primary">
                    <HugeiconsIcon icon={Sent02Icon} className="size-4" strokeWidth={2} />
                  </span>
                </span>
              ) : null}
              <GradualScoreNumber
                key={`total-${phase}`}
                from={isCountingDown ? roundScore.displayedTotal : previousScore.displayedTotal}
                value={score.displayedTotal}
                isCounting={isCountingUp || isCountingDown}
                duration={isCountingDown ? SCORE_ADJUSTMENT_COUNT_DURATION_MS : undefined}
                trend={isCountingDown ? -1 : 1}
              />
            </span>
          </P>
        </div>
      </TableCell>
      <TableCell className="text-right">
        <Caption
          data-round-value-slot
          className={cn(
            "inline-block w-[4ch] text-right",
            score.displayedGained > 0 && "font-medium text-primary",
          )}
        >
          <span data-score-gained>
            <GradualScoreNumber
              key={`gained-${phase}`}
              from={previousScore.displayedGained}
              value={score.displayedGained}
              isCounting={isCountingUp}
              prefix={score.displayedGained > 0 ? "+" : undefined}
            />
          </span>
        </Caption>
      </TableCell>
    </>
  );
}

function WinnerCrown({ large = false }: { large?: boolean }) {
  const shouldReduceMotion = useShouldReduceMotion();

  return (
    <motion.span
      aria-hidden="true"
      className={cn(
        "absolute z-20 origin-bottom-left drop-shadow-sm",
        large ? "-top-9 right-1 text-5xl/none" : "-top-4 -right-0.5 rotate-18 text-sm/none",
      )}
      style={{ fontFamily: '"Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji"' }}
      initial={{ opacity: 0, y: shouldReduceMotion ? 0 : -16, rotate: large ? -12 : 2, scale: 0.5 }}
      animate={{ opacity: 1, y: 0, rotate: large ? 8 : 18, scale: 1 }}
      transition={{
        delay: shouldReduceMotion ? 0 : large ? 0.45 : 0.16,
        type: "spring",
        stiffness: 420,
        damping: 17,
      }}
    >
      👑
    </motion.span>
  );
}

function GameWinnerTakeover({
  winner,
  onComplete,
}: {
  winner: PlayerSnapshot;
  onComplete: () => void;
}) {
  const shouldReduceMotion = useShouldReduceMotion();
  const completeTakeover = useEffectEvent(onComplete);

  useEffect(() => {
    void fireStreamingCelebrationConfetti({ durationMs: 3_200, delayMs: 260 });
    const timer = window.setTimeout(
      completeTakeover,
      shouldReduceMotion ? 2_000 : GAME_WINNER_DURATION_MS,
    );

    return () => window.clearTimeout(timer);
  }, [shouldReduceMotion]);

  return (
    <motion.div
      data-slot="game-winner-takeover"
      className="fixed inset-0 z-100 grid place-items-center overflow-hidden bg-[radial-gradient(circle_at_50%_42%,color-mix(in_oklab,var(--primary)_28%,var(--background)),var(--background)_62%)] p-6 text-center"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: shouldReduceMotion ? 0.15 : 0.38 }}
      role="status"
      aria-live="assertive"
      aria-label={m.wins_game({ name: winner.name })}
    >
      <motion.div
        className="relative z-10 flex flex-col items-center"
        initial={{ y: shouldReduceMotion ? 0 : 28, scale: shouldReduceMotion ? 1 : 0.82 }}
        animate={{ y: 0, scale: 1 }}
        transition={{ delay: 0.12, type: "spring", stiffness: 240, damping: 20 }}
      >
        <Caption className="mb-8 font-semibold tracking-[0.32em] text-primary uppercase">
          {m.match_champion()}
        </Caption>
        <div className="relative mb-7">
          <Avatar className="size-28 border-4 border-background shadow-2xl ring-4 ring-primary/30 md:size-36">
            {winner.imageUrl ? <AvatarImage src={winner.imageUrl} alt={winner.name} /> : null}
            <AvatarFallback className="text-3xl md:text-4xl">
              {getUserInitials(winner.name)}
            </AvatarFallback>
          </Avatar>
          <WinnerCrown large />
        </div>
        <motion.h1
          className="font-heading text-4xl/none font-bold tracking-tighter text-balance md:text-7xl/none"
          initial={{ opacity: 0, y: shouldReduceMotion ? 0 : 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: shouldReduceMotion ? 0 : 0.42, duration: 0.45 }}
        >
          {winner.name}
        </motion.h1>
        <P className="mt-4 text-muted-foreground">{m.wins_game({ name: winner.name })}</P>
        <Button
          type="button"
          variant="secondary"
          className="pointer-events-auto mt-10"
          onClick={onComplete}
        >
          {m.final_scores()}
        </Button>
      </motion.div>
    </motion.div>
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

type RankingRow = ReturnType<typeof rankingRows>[number];
type ScoreRevealState = {
  key: string;
  phaseByPlayerId: Record<string, ResultScoreTimelinePhase>;
  hasReordered: boolean;
};

function scoreTimeline(game: GameSnapshot, players: PlayerSnapshot[]) {
  const timeline: { playerId: string; hasAdjustment: boolean }[] = [];

  for (const { playerState } of rankingRows(game, players, "previous")) {
    const previous = resultScoreState(playerState, "previous");
    const round = resultScoreState(playerState, "round");
    const adjusted = resultScoreState(playerState, "adjusted");

    if (
      previous.displayedTotal !== round.displayedTotal ||
      round.displayedTotal !== adjusted.displayedTotal
    ) {
      timeline.push({
        playerId: playerState.playerId,
        hasAdjustment: round.displayedTotal !== adjusted.displayedTotal,
      });
    }
  }

  return timeline;
}

function scoreValuesKey(gamePlayers: GameSnapshot["players"]) {
  let key = "";

  for (const player of gamePlayers) {
    key += `${player.playerId}:${player.totalPoints}:${player.pointsGained}:${player.unadjustedTotalPoints ?? ""}|`;
  }

  return key;
}

function adjustedScorePhases(gamePlayers: GameSnapshot["players"]) {
  const phases: Record<string, ResultScoreTimelinePhase> = {};

  for (const player of gamePlayers) phases[player.playerId] = "adjusted";

  return phases;
}

function useResultScoreReveal({
  game,
  players,
  shouldReduceMotion,
  winnerTakeover,
}: {
  game: GameSnapshot;
  players: PlayerSnapshot[];
  shouldReduceMotion: boolean;
  winnerTakeover: boolean;
}) {
  const scoreRevealKey = `${game.round}:${shouldReduceMotion}`;
  const scoreTimelineKey = `${scoreRevealKey}:${scoreValuesKey(game.players)}:${winnerTakeover ? "paused" : "active"}`;
  const [scoreReveal, setScoreReveal] = useState<ScoreRevealState>(() => ({
    key: scoreTimelineKey,
    phaseByPlayerId: {},
    hasReordered: false,
  }));
  const scorePlayerIds = new Set<string>();

  for (const { playerId: scorePlayerId } of scoreTimeline(game, players)) {
    scorePlayerIds.add(scorePlayerId);
  }

  const revealState: ScoreRevealState = winnerTakeover
    ? { key: scoreTimelineKey, phaseByPlayerId: {}, hasReordered: false }
    : shouldReduceMotion
      ? {
          key: scoreTimelineKey,
          phaseByPlayerId: adjustedScorePhases(game.players),
          hasReordered: true,
        }
      : scoreReveal.key === scoreTimelineKey
        ? scoreReveal
        : { key: scoreTimelineKey, phaseByPlayerId: {}, hasReordered: false };

  const scheduleScoreReveal = useEffectEvent((activeScoreTimelineKey: string) => {
    const timers: number[] = [];
    let nextRevealAt = SCORE_REVEAL_DELAY_MS;
    let finalScoreAt = SCORE_REVEAL_DELAY_MS;

    for (const { playerId: scorePlayerId, hasAdjustment } of scoreTimeline(game, players)) {
      const revealAt = nextRevealAt;
      const roundAt = revealAt + SCORE_COUNT_DURATION_MS;
      const adjustingAt = roundAt + SCORE_ADJUSTMENT_HOLD_MS;
      const adjustedAt = hasAdjustment
        ? adjustingAt + SCORE_ADJUSTMENT_COUNT_DURATION_MS
        : revealAt + SCORE_COUNT_DURATION_MS + SCORE_ADJUSTMENT_HOLD_MS;

      timers.push(
        window.setTimeout(() => {
          setScoreReveal((current) => ({
            key: activeScoreTimelineKey,
            phaseByPlayerId: {
              ...(current.key === activeScoreTimelineKey ? current.phaseByPlayerId : {}),
              [scorePlayerId]: "counting",
            },
            hasReordered: false,
          }));
        }, revealAt),
      );
      timers.push(
        window.setTimeout(() => {
          setScoreReveal((current) => ({
            key: activeScoreTimelineKey,
            phaseByPlayerId: {
              ...(current.key === activeScoreTimelineKey ? current.phaseByPlayerId : {}),
              [scorePlayerId]: "round",
            },
            hasReordered: false,
          }));
        }, roundAt),
      );
      if (hasAdjustment) {
        timers.push(
          window.setTimeout(() => {
            setScoreReveal((current) => ({
              key: activeScoreTimelineKey,
              phaseByPlayerId: {
                ...(current.key === activeScoreTimelineKey ? current.phaseByPlayerId : {}),
                [scorePlayerId]: "adjusting",
              },
              hasReordered: false,
            }));
          }, adjustingAt),
        );
      }
      timers.push(
        window.setTimeout(() => {
          setScoreReveal((current) => ({
            key: activeScoreTimelineKey,
            phaseByPlayerId: {
              ...(current.key === activeScoreTimelineKey ? current.phaseByPlayerId : {}),
              [scorePlayerId]: "adjusted",
            },
            hasReordered: false,
          }));
        }, adjustedAt),
      );

      finalScoreAt = adjustedAt;
      nextRevealAt = hasAdjustment
        ? adjustedAt + SCORE_REVEAL_HANDOFF_MS
        : revealAt + SCORE_REVEAL_STEP_MS;
    }

    timers.push(
      window.setTimeout(() => {
        setScoreReveal({
          key: activeScoreTimelineKey,
          phaseByPlayerId: adjustedScorePhases(game.players),
          hasReordered: true,
        });
      }, finalScoreAt + RANKING_REORDER_DELAY_MS),
    );

    return () => {
      for (const timer of timers) window.clearTimeout(timer);
    };
  });

  useEffect(() => {
    if (winnerTakeover || shouldReduceMotion) return;

    return scheduleScoreReveal(scoreTimelineKey);
  }, [scoreTimelineKey, shouldReduceMotion, winnerTakeover]);

  return { scorePlayerIds, scoreRevealKey, revealState };
}

function ResultsScoreTable({
  rows,
  revealState,
  scorePlayerIds,
  shouldReduceMotion,
}: {
  rows: RankingRow[];
  revealState: ScoreRevealState;
  scorePlayerIds: Set<string>;
  shouldReduceMotion: boolean;
}) {
  return (
    <Table containerClassName="rounded-lg border border-border/70">
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
        {rows.map(({ rank, player, playerState, isRoundWinner }) => {
          const playerName = player?.name ?? m.unknown_player();
          const scorePhase = shouldReduceMotion
            ? "adjusted"
            : scorePlayerIds.has(playerState.playerId)
              ? (revealState.phaseByPlayerId[playerState.playerId] ?? "previous")
              : "adjusted";

          return (
            <motion.tr
              key={playerState.playerId}
              layout="position"
              data-slot="result-row"
              data-player-id={playerState.playerId}
              data-rankings-reordered={revealState.hasReordered || undefined}
              className={cn(
                "border-b transition-colors last:border-0 hover:bg-muted/50",
                isRoundWinner && "border-primary/35 bg-primary/10 hover:bg-primary/15",
              )}
              transition={{ type: "spring", stiffness: 380, damping: 32 }}
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
                    {isRoundWinner ? <WinnerCrown /> : null}
                  </Avatar>
                  <P size="sm" className={cn("font-medium", isRoundWinner && "text-primary")}>
                    {playerName}
                  </P>
                </div>
              </TableCell>
              <TableCell className="text-right">
                {playerState.hand && playerState.hand.length !== 0 ? (
                  <LeftoverHandTooltip
                    handCount={playerState.handCount}
                    hand={playerState.hand}
                    playerName={playerName}
                  />
                ) : null}
              </TableCell>
              <ResultPoints playerState={playerState} phase={scorePhase} />
            </motion.tr>
          );
        })}
      </TableBody>
    </Table>
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
  const celebratedRoundRef = useRef<string | null>(null);
  const isWinner = winner?.playerId === playerId;
  const shouldReduceMotion = useShouldReduceMotion();
  const winnerTakeoverKey =
    isGameOver && winner ? `${room?.code ?? "game"}:${game.round}:${winner.playerId}` : null;
  const [dismissedWinnerTakeoverKey, setDismissedWinnerTakeoverKey] = useState<string | null>(null);
  const winnerTakeover = Boolean(
    winnerTakeoverKey && winnerTakeoverKey !== dismissedWinnerTakeoverKey,
  );
  const { scorePlayerIds, scoreRevealKey, revealState } = useResultScoreReveal({
    game,
    players,
    shouldReduceMotion,
    winnerTakeover,
  });
  const visibleRows = rankingRows(
    game,
    players,
    shouldReduceMotion || revealState.hasReordered ? "adjusted" : "previous",
  );
  const completeWinnerTakeover = () => {
    setDismissedWinnerTakeoverKey(winnerTakeoverKey);
  };

  useEffect(() => {
    if (isGameOver || !isWinner || celebratedRoundRef.current === scoreRevealKey) return;

    celebratedRoundRef.current = scoreRevealKey;
    void fireCelebrationConfetti({ delayMs: 250 });
  }, [isGameOver, isWinner, scoreRevealKey]);

  return (
    <>
      <AnimatePresence>
        {winnerTakeover && winner ? (
          <GameWinnerTakeover
            key={winnerTakeoverKey ?? winner.playerId}
            winner={winner}
            onComplete={completeWinnerTakeover}
          />
        ) : null}
      </AnimatePresence>

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
            <ResultsScoreTable
              rows={visibleRows}
              revealState={revealState}
              scorePlayerIds={scorePlayerIds}
              shouldReduceMotion={shouldReduceMotion}
            />

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
    </>
  );
}

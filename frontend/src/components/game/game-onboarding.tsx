import {
  createContext,
  useContext,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from "react";
import {
  ArrowRight01Icon,
  CheckmarkCircle02Icon,
  JokerIcon,
  SparklesIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { AnimatePresence, motion } from "motion/react";
import { toast } from "sonner";
import { GameBoardView } from "#/components/game/game-board-view";
import {
  type ActionResult,
  type CardSnapshot,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";
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
import { authClient } from "#/lib/auth-client";
import { fireCelebrationConfetti } from "#/lib/confetti";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

const TUTORIAL_PLAYER_ID = "tutorial-player";
const TUTORIAL_DRAW_CARD: CardSnapshot = { rank: 7, suit: 3 };
const TUTORIAL_STAGES = ["intro", "orientation", "draw", "compose", "discard", "complete"] as const;
const ORIENTATION_HAND_CARDS_SELECTOR =
  '[data-onboarding-target="hand"] [data-onboarding-target="hand-cards"]';
const ORIENTATION_PILES_SELECTOR =
  '[data-card-motion-source="deck"],[data-card-motion-source="discard"]';
const ORIENTATION_COMPOSITION_SELECTOR = '[data-onboarding-target="table"]';
const COMPOSE_HAND_CARD_SELECTORS = [5, 6, 7].map(
  (rank) => `[data-onboarding-target="hand"] [data-card-rank="${rank}"][data-card-suit="3"]`,
);
const COMPACT_TUTORIAL_QUERY = "(max-width: 47.999rem), (max-height: 42.5rem)";

type TutorialStage = (typeof TUTORIAL_STAGES)[number];
type GameOnboardingContextValue = {
  startTutorial: () => void;
};

const GameOnboardingContext = createContext<GameOnboardingContextValue | null>(null);

function subscribeToCompactTutorialLayout(onChange: () => void) {
  if (typeof window === "undefined") return () => {};
  const mediaQuery = window.matchMedia?.(COMPACT_TUTORIAL_QUERY);
  mediaQuery?.addEventListener("change", onChange);
  return () => mediaQuery?.removeEventListener("change", onChange);
}

function isCompactTutorialLayout() {
  return (
    typeof window !== "undefined" && window.matchMedia?.(COMPACT_TUTORIAL_QUERY).matches === true
  );
}

function useCompactTutorialLayout() {
  return useSyncExternalStore(
    subscribeToCompactTutorialLayout,
    isCompactTutorialLayout,
    () => false,
  );
}

export function useGameOnboarding() {
  const context = useContext(GameOnboardingContext);
  if (!context) {
    throw new Error("useGameOnboarding must be used within GameOnboardingProvider");
  }
  return context;
}

type SpotlightRect = {
  id: string;
  top: number;
  right: number;
  bottom: number;
  left: number;
  width: number;
  height: number;
  cornerRadius?: number;
};

const tutorialPlayers: PlayerSnapshot[] = [
  {
    playerId: TUTORIAL_PLAYER_ID,
    name: "Alex",
    connected: true,
    seat: 0,
    isHost: true,
    canReconnect: false,
  },
  {
    playerId: "tutorial-opponent",
    name: "Mira",
    connected: true,
    seat: 1,
    isHost: false,
    canReconnect: false,
  },
];

function createTutorialGame(): GameSnapshot {
  return {
    gameMode: "quick",
    phase: 1,
    round: 1,
    dealerIndex: 1,
    roundWinnerIndex: -1,
    turn: {
      number: 3,
      playerIndex: 0,
      playerId: TUTORIAL_PLAYER_ID,
      hasDrawn: false,
      mustUseDiscardDraw: false,
    },
    players: [
      {
        playerId: TUTORIAL_PLAYER_ID,
        handCount: 3,
        totalPoints: 0,
        pointsGained: 0,
        hasOpened: true,
      },
      {
        playerId: "tutorial-opponent",
        handCount: 7,
        totalPoints: 18,
        pointsGained: 18,
        hasOpened: true,
      },
    ],
    hand: [
      { rank: 5, suit: 3 },
      { rank: 6, suit: 3 },
      { rank: 12, suit: 0 },
    ],
    drawPileCount: 18,
    discardPile: [
      { rank: 4, suit: 1 },
      { rank: 11, suit: 2 },
    ],
    activeCompositions: [
      {
        type: "set",
        cards: [
          { rank: 9, suit: 0 },
          { rank: 9, suit: 1 },
          { rank: 9, suit: 2 },
        ],
        points: 27,
        complete: false,
      },
    ],
  };
}

function cardsMatch(left: CardSnapshot, right: CardSnapshot) {
  return (
    Boolean(left.isJoker) === Boolean(right.isJoker) &&
    left.rank === right.rank &&
    left.suit === right.suit
  );
}

function removeCards(hand: CardSnapshot[], cards: CardSnapshot[]) {
  const nextHand = [...hand];
  for (const card of cards) {
    const index = nextHand.findIndex((candidate) => cardsMatch(candidate, card));
    if (index >= 0) nextHand.splice(index, 1);
  }
  return nextHand;
}

function compositionPoints(cards: CardSnapshot[]) {
  return cards.reduce(
    (total, card) => total + (card.isJoker ? 20 : Math.min(card.rank ?? 0, 10)),
    0,
  );
}

function isTutorialRun(cards: CardSnapshot[]) {
  if (cards.length !== 3) return false;
  return [5, 6, 7].every((rank) =>
    cards.some((card) => !card.isJoker && card.rank === rank && card.suit === 3),
  );
}

function isTutorialDraftComplete(drafts: DraftCompositionSnapshot[]) {
  return (
    drafts.length === 1 && drafts[0]?.tableIndex === undefined && isTutorialRun(drafts[0].cards)
  );
}

function tutorialDraftProgress(drafts: DraftCompositionSnapshot[]) {
  let maxProgress = 0;
  for (const draft of drafts) {
    if (draft.tableIndex !== undefined) continue;

    const ranks = new Set<number>();
    for (const card of draft.cards) {
      if (!card.isJoker && card.suit === 3 && card.rank && [5, 6, 7].includes(card.rank)) {
        ranks.add(card.rank);
      }
    }
    maxProgress = Math.max(maxProgress, ranks.size);
  }
  return maxProgress;
}

function selectorsForStage(stage: TutorialStage, isDrawingCard: boolean) {
  switch (stage) {
    case "orientation":
      return [
        ORIENTATION_HAND_CARDS_SELECTOR,
        ORIENTATION_PILES_SELECTOR,
        ORIENTATION_COMPOSITION_SELECTOR,
      ];
    case "draw":
      return isDrawingCard
        ? ['[data-card-motion-source="deck"]', '[data-onboarding-target="hand"]']
        : ['[data-card-motion-source="deck"]'];
    case "compose":
      return [...COMPOSE_HAND_CARD_SELECTORS, '[data-onboarding-target="new-composition"]'];
    case "discard":
      return [
        '[data-onboarding-target="hand"] [data-card-rank="12"][data-card-suit="0"]',
        '[data-card-motion-source="discard"]',
      ];
    default:
      return [];
  }
}

function useSpotlightRect(
  rootRef: RefObject<HTMLElement | null>,
  stage: TutorialStage,
  isDrawingCard: boolean,
) {
  const [rects, setRects] = useState<SpotlightRect[]>([]);

  useLayoutEffect(() => {
    const root = rootRef.current;
    const selectors = selectorsForStage(stage, isDrawingCard);
    if (!root || selectors.length === 0) {
      setRects([]);
      return;
    }
    const spotlightRoot = root;

    const requestFrame =
      window.requestAnimationFrame?.bind(window) ??
      ((callback: FrameRequestCallback) => window.setTimeout(callback, 0));
    const cancelFrame =
      window.cancelAnimationFrame?.bind(window) ??
      ((frameId: number) => window.clearTimeout(frameId));
    let frame = 0;
    let trackingFrame = 0;
    let trackUntil = 0;

    function measure() {
      cancelFrame(frame);
      frame = requestFrame(() => {
        const targetBounds: Array<{ id: string; bounds: DOMRect[]; cornerRadius: number }> = [];
        for (const selector of selectors) {
          const bounds: DOMRect[] = [];
          let cornerRadius = 0;
          for (const element of spotlightRoot.querySelectorAll(selector)) {
            const rect = element.getBoundingClientRect();
            if (rect.width > 0 && rect.height > 0) {
              bounds.push(rect);
              const style = window.getComputedStyle(element);
              cornerRadius = Math.max(
                cornerRadius,
                Number.parseFloat(style.borderTopLeftRadius) || 0,
                Number.parseFloat(style.borderTopRightRadius) || 0,
                Number.parseFloat(style.borderBottomRightRadius) || 0,
                Number.parseFloat(style.borderBottomLeftRadius) || 0,
                Number.parseFloat(style.borderRadius) || 0,
              );
            }
          }
          if (bounds.length > 0) targetBounds.push({ id: selector, bounds, cornerRadius });
        }
        if (targetBounds.length === 0) {
          setRects([]);
          return;
        }

        const padding = 5;
        const nextRects: SpotlightRect[] = [];
        for (const target of targetBounds) {
          let minLeft = Number.POSITIVE_INFINITY;
          let minTop = Number.POSITIVE_INFINITY;
          let maxRight = Number.NEGATIVE_INFINITY;
          let maxBottom = Number.NEGATIVE_INFINITY;
          for (const rect of target.bounds) {
            minLeft = Math.min(minLeft, rect.left);
            minTop = Math.min(minTop, rect.top);
            maxRight = Math.max(maxRight, rect.right);
            maxBottom = Math.max(maxBottom, rect.bottom);
          }

          const left = Math.max(4, minLeft - padding);
          const top = Math.max(4, minTop - padding);
          const right = Math.min(window.innerWidth - 4, maxRight + padding);
          const bottom = Math.min(window.innerHeight - 4, maxBottom + padding);
          nextRects.push({
            id: target.id,
            top,
            right,
            bottom,
            left,
            width: right - left,
            height: bottom - top,
            cornerRadius: Math.max(18, target.cornerRadius + padding),
          });
        }
        setRects(nextRects);
      });
    }

    function trackLayoutMotion() {
      trackUntil = performance.now() + 350;
      if (trackingFrame) return;

      function trackFrame() {
        measure();
        trackingFrame = performance.now() < trackUntil ? requestFrame(trackFrame) : 0;
      }

      trackingFrame = requestFrame(trackFrame);
    }

    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(measure);
    observer?.observe(spotlightRoot);
    const mutationObserver =
      typeof MutationObserver === "undefined"
        ? null
        : new MutationObserver(stage === "compose" ? trackLayoutMotion : measure);
    mutationObserver?.observe(spotlightRoot, {
      childList: true,
      subtree: true,
      attributes: stage === "compose",
      attributeFilter: stage === "compose" ? ["class", "style"] : undefined,
    });
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    measure();

    return () => {
      cancelFrame(frame);
      cancelFrame(trackingFrame);
      observer?.disconnect();
      mutationObserver?.disconnect();
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure, true);
    };
  }, [rootRef, stage, isDrawingCard]);

  return rects;
}

const backdropClassName =
  "pointer-events-none fixed z-20 bg-black/30 supports-backdrop-filter:backdrop-blur-sm";

function SpotlightBackdrop({ rects }: { rects: SpotlightRect[] }) {
  const maskId = `onboarding-spotlight-mask-${useId().replaceAll(":", "")}`;

  if (rects.length === 0) {
    return <div data-onboarding-backdrop className={cn(backdropClassName, "inset-0")} />;
  }

  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  const maskImage = `url(#${maskId})`;

  return (
    <>
      <svg data-onboarding-mask className="pointer-events-none fixed size-0" aria-hidden="true">
        <defs>
          <mask
            id={maskId}
            maskUnits="userSpaceOnUse"
            x="0"
            y="0"
            width={viewportWidth}
            height={viewportHeight}
          >
            <rect width={viewportWidth} height={viewportHeight} fill="white" />
            {rects.map((rect) => (
              <rect
                key={rect.id}
                data-onboarding-mask-cutout
                x={rect.left}
                y={rect.top}
                width={rect.width}
                height={rect.height}
                rx={Math.min(rect.cornerRadius ?? 18, rect.width / 2, rect.height / 2)}
                fill="black"
              />
            ))}
          </mask>
        </defs>
      </svg>
      <div
        data-onboarding-backdrop
        className={cn(backdropClassName, "inset-0")}
        style={{
          WebkitMaskImage: maskImage,
          maskImage,
        }}
      />
      {rects.map((rect) => (
        <motion.div
          key={rect.id}
          data-onboarding-spotlight
          layout
          className="pointer-events-none fixed z-30 ring-2 ring-primary/80 ring-offset-2 ring-offset-transparent"
          initial={false}
          style={{
            top: rect.top,
            left: rect.left,
            width: rect.width,
            height: rect.height,
            borderRadius: rect.cornerRadius ?? 18,
          }}
          transition={{ layout: { duration: 0.2, ease: [0.23, 1, 0.32, 1] } }}
        />
      ))}
    </>
  );
}

function orientationLabel(id: string) {
  switch (id) {
    case ORIENTATION_HAND_CARDS_SELECTOR:
      return m.onboarding_orientation_hand();
    case ORIENTATION_PILES_SELECTOR:
      return m.onboarding_orientation_piles();
    case ORIENTATION_COMPOSITION_SELECTOR:
      return m.onboarding_orientation_composition();
    default:
      return null;
  }
}

function OrientationLabels({ rects }: { rects: SpotlightRect[] }) {
  return (
    <div className="pointer-events-none fixed inset-0 z-40" aria-hidden="true">
      {rects.map((rect, index) => {
        const label = orientationLabel(rect.id);
        if (!label) return null;
        const left = `min(max(8px, ${rect.left + 24}px), calc(100vw - 168px))`;
        const top = rect.top - 3;

        return (
          <motion.div
            key={rect.id}
            data-onboarding-orientation-label
            className="fixed flex h-0 items-center"
            style={{ top, left }}
            initial={{ opacity: 0, x: -4 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.05, duration: 0.18 }}
          >
            <Badge className="max-w-40 truncate shadow-md ring-4 ring-background">{label}</Badge>
          </motion.div>
        );
      })}
    </div>
  );
}

function orientationCoachPosition(rect: SpotlightRect | null): CSSProperties {
  if (!rect || typeof window === "undefined") {
    return { top: 16, right: 16, width: 384 };
  }

  const margin = 16;
  const inset = 20;
  const width = Math.min(384, window.innerWidth - margin * 2);
  const estimatedHeight = 210;

  return {
    top: Math.min(
      Math.max(margin, rect.top + inset),
      window.innerHeight - estimatedHeight - margin,
    ),
    left: Math.min(
      Math.max(margin, rect.right - width - inset),
      window.innerWidth - width - margin,
    ),
    width,
  };
}

function coachPosition(rect: SpotlightRect | null): CSSProperties {
  if (!rect || typeof window === "undefined") {
    return { top: "50%", left: "50%", transform: "translate(-50%, -50%)" };
  }

  const gap = 12;
  const margin = 16;
  const width = Math.min(384, window.innerWidth - margin * 2);
  const estimatedHeight = 210;
  const spaceRight = window.innerWidth - rect.right;
  const spaceLeft = rect.left;
  const spaceBelow = window.innerHeight - rect.bottom;
  const spaceAbove = rect.top;

  if (spaceRight >= width + gap) {
    return {
      top: Math.min(Math.max(margin, rect.top), window.innerHeight - estimatedHeight - margin),
      left: rect.right + gap,
      width,
    };
  }
  if (spaceLeft >= width + gap) {
    return {
      top: Math.min(Math.max(margin, rect.top), window.innerHeight - estimatedHeight - margin),
      left: rect.left - width - gap,
      width,
    };
  }
  if (spaceBelow >= estimatedHeight + gap) {
    return {
      top: rect.bottom + gap,
      left: Math.min(Math.max(margin, rect.left), window.innerWidth - width - margin),
      width,
    };
  }
  if (spaceAbove >= estimatedHeight + gap) {
    return {
      top: rect.top - estimatedHeight - gap,
      left: Math.min(Math.max(margin, rect.left), window.innerWidth - width - margin),
      width,
    };
  }

  return { top: margin, left: margin, width };
}

function stageContent(stage: TutorialStage) {
  switch (stage) {
    case "intro":
      return {
        eyebrow: m.onboarding_welcome_eyebrow(),
        title: m.onboarding_welcome_title(),
        description: m.onboarding_welcome_description(),
      };
    case "orientation":
      return {
        eyebrow: m.onboarding_orientation_eyebrow(),
        title: m.onboarding_orientation_title(),
        description: m.onboarding_orientation_description(),
      };
    case "draw":
      return {
        eyebrow: m.onboarding_draw_eyebrow(),
        title: m.onboarding_draw_title(),
        description: m.onboarding_draw_hint(),
      };
    case "compose":
      return {
        eyebrow: m.onboarding_compose_eyebrow(),
        title: m.onboarding_compose_title(),
        description: m.onboarding_compose_hint(),
      };
    case "discard":
      return {
        eyebrow: m.onboarding_discard_eyebrow(),
        title: m.onboarding_discard_title(),
        description: m.onboarding_discard_hint(),
      };
    case "complete":
      return {
        eyebrow: m.onboarding_complete_eyebrow(),
        title: m.onboarding_complete_title(),
        description: m.onboarding_complete_description(),
      };
  }
}

function TutorialCoach({
  stage,
  rect,
  compositionProgress,
  isCompleting,
  onContinue,
  presentation,
}: {
  stage: TutorialStage;
  rect: SpotlightRect | null;
  compositionProgress: number;
  isCompleting: boolean;
  onContinue: () => void;
  presentation: "inline" | "overlay";
}) {
  const content = stageContent(stage);
  const stageIndex = TUTORIAL_STAGES.indexOf(stage);
  const hasContinue = stage === "intro" || stage === "orientation" || stage === "complete";
  const isInline = presentation === "inline";
  const isCentered = !isInline && (stage === "intro" || stage === "complete");

  return (
    <div
      className={cn(
        isInline ? "relative z-40 shrink-0 px-2 pb-2" : "pointer-events-none fixed inset-0 z-40",
      )}
    >
      <AnimatePresence mode="wait">
        <motion.div
          key={stage}
          data-centered={isCentered || undefined}
          data-tutorial-stage={stage}
          data-tutorial-presentation={presentation}
          className={cn(
            "pointer-events-none",
            isInline ? "relative" : "fixed",
            isCentered ? "inset-0 grid place-items-center p-4" : null,
          )}
          style={
            isInline || isCentered
              ? undefined
              : stage === "orientation"
                ? orientationCoachPosition(rect)
                : coachPosition(rect)
          }
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.18, ease: [0.23, 1, 0.32, 1] }}
        >
          <motion.div
            className={cn(
              "pointer-events-auto",
              isInline ? "mx-auto w-full max-w-2xl" : "w-[min(24rem,calc(100vw-2rem))]",
            )}
            initial={{ y: 8, scale: 0.98 }}
            animate={{ y: 0, scale: 1 }}
            exit={{ y: -6, scale: 0.98 }}
            transition={{ duration: 0.18, ease: [0.23, 1, 0.32, 1] }}
          >
            <Card
              size="sm"
              role="dialog"
              aria-modal={isCentered ? "true" : undefined}
              aria-labelledby="tutorial-coach-title"
              aria-describedby="tutorial-coach-description"
              className={cn(
                "ring-1 ring-foreground/10",
                isInline
                  ? "gap-2 rounded-3xl py-2 shadow-lg [--card-spacing:--spacing(3)]"
                  : "shadow-2xl",
              )}
            >
              <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
                <div
                  className="flex min-w-0 items-center gap-1.5"
                  aria-label={m.onboarding_progress()}
                >
                  {TUTORIAL_STAGES.map((item, index) => (
                    <span
                      key={item}
                      className={cn(
                        "h-1.5 rounded-full",
                        index === stageIndex ? "w-7 bg-primary" : "w-1.5 bg-border",
                      )}
                      aria-current={index === stageIndex ? "step" : undefined}
                    />
                  ))}
                </div>
                <Badge variant="secondary" className="shrink-0 whitespace-nowrap">
                  {stage === "intro" ? (
                    <HugeiconsIcon icon={SparklesIcon} data-icon="inline-start" />
                  ) : stage === "complete" ? (
                    <HugeiconsIcon icon={CheckmarkCircle02Icon} data-icon="inline-start" />
                  ) : (
                    <span className="size-1.5 rounded-full bg-primary" aria-hidden="true" />
                  )}
                  {content.eyebrow}
                </Badge>
              </CardHeader>
              <CardContent className={cn("grid gap-1.5", isInline ? "gap-0.5" : null)}>
                <CardTitle
                  id="tutorial-coach-title"
                  className={isInline ? "text-base/5" : "text-lg/6"}
                >
                  {content.title}
                </CardTitle>
                <CardDescription
                  id="tutorial-coach-description"
                  className={isInline ? "text-xs/4" : undefined}
                >
                  {content.description}
                </CardDescription>
              </CardContent>
              <CardFooter className="flex-wrap justify-end gap-2">
                {hasContinue ? (
                  <Button type="button" onClick={onContinue} disabled={isCompleting}>
                    {stage === "complete"
                      ? m.onboarding_done()
                      : stage === "orientation"
                        ? m.onboarding_orientation_done()
                        : m.onboarding_begin()}
                    {stage === "intro" || stage === "orientation" ? (
                      <HugeiconsIcon icon={ArrowRight01Icon} data-icon="inline-end" />
                    ) : null}
                  </Button>
                ) : stage === "compose" ? (
                  <Badge variant="outline">
                    {m.onboarding_cards_placed({ count: compositionProgress, total: 3 })}
                  </Badge>
                ) : (
                  <Badge variant="outline">{m.onboarding_waiting_for_action()}</Badge>
                )}
              </CardFooter>
            </Card>
          </motion.div>
        </motion.div>
      </AnimatePresence>
    </div>
  );
}

function TutorialGame({
  isCompleting,
  onComplete,
}: {
  isCompleting: boolean;
  onComplete: () => Promise<void>;
}) {
  const boardRef = useRef<HTMLDivElement>(null);
  const [stage, setStage] = useState<TutorialStage>("intro");
  const [game, setGame] = useState(createTutorialGame);
  const [compositionProgress, setCompositionProgress] = useState(0);
  const [isDrawingCard, setIsDrawingCard] = useState(false);
  const shouldReduceMotion = useShouldReduceMotion();
  const compactLayout = useCompactTutorialLayout();
  const spotlightRects = useSpotlightRect(boardRef, stage, isDrawingCard);
  const spotlightRect = (stage === "draw" ? spotlightRects.at(0) : spotlightRects.at(-1)) ?? null;
  const canCompose = stage === "compose" || stage === "discard";
  const useInlineCoach = compactLayout && stage !== "intro" && stage !== "complete";

  function continueTutorial() {
    if (stage === "complete") {
      void onComplete();
      return;
    }
    setStage(stage === "intro" ? "orientation" : "draw");
  }

  function drawFromDeck() {
    setGame((current) => {
      if (current.turn.hasDrawn) return current;
      const nextHand = [...current.hand, TUTORIAL_DRAW_CARD];
      return {
        ...current,
        hand: nextHand,
        drawPileCount: current.drawPileCount - 1,
        turn: { ...current.turn, hasDrawn: true },
        players: current.players.map((player) =>
          player.playerId === TUTORIAL_PLAYER_ID
            ? { ...player, handCount: nextHand.length }
            : player,
        ),
      };
    });
  }

  async function playTableAndDiscard(
    play: TablePlayRequest,
    _cardIndex: number,
    discardedCard: CardSnapshot,
  ): Promise<ActionResult> {
    const hasExpectedPlay =
      stage === "discard" &&
      play.compositions.length === 1 &&
      play.additions.length === 0 &&
      play.reclaims.length === 0 &&
      isTutorialRun(play.compositions[0]?.cards ?? []) &&
      cardsMatch(discardedCard, { rank: 12, suit: 0 });

    if (!hasExpectedPlay) {
      return { action: "play_and_discard", playerId: TUTORIAL_PLAYER_ID, ok: false };
    }

    setGame((current) => {
      const playedCards = play.compositions.flatMap((composition) => composition.cards);
      const remainingHand = removeCards(removeCards(current.hand, playedCards), [discardedCard]);
      return {
        ...current,
        hand: remainingHand,
        discardPile: [discardedCard, ...current.discardPile],
        activeCompositions: [
          ...current.activeCompositions,
          ...play.compositions.map((composition) => ({
            type: "run",
            cards: composition.cards,
            points: compositionPoints(composition.cards),
            complete: false,
          })),
        ],
        players: current.players.map((player) =>
          player.playerId === TUTORIAL_PLAYER_ID
            ? { ...player, handCount: remainingHand.length }
            : player,
        ),
      };
    });
    setStage("complete");
    void fireCelebrationConfetti({ count: 140, originY: 0.72, delayMs: 100 });
    return { action: "play_and_discard", playerId: TUTORIAL_PLAYER_ID, ok: true };
  }

  return (
    <motion.section
      className="fixed inset-0 z-40 flex min-h-0 flex-col bg-background p-2"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: shouldReduceMotion ? 0.08 : 0.2 }}
      aria-label={m.onboarding_practice_game()}
    >
      <header className="flex h-12 shrink-0 items-center gap-2 px-2 md:h-14 md:px-3">
        <span className="grid size-8 place-items-center rounded-full bg-primary/10 text-primary">
          <HugeiconsIcon icon={JokerIcon} aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="truncate font-heading text-sm font-semibold">
            {m.onboarding_practice_game()}
          </p>
          <p className="text-xs text-muted-foreground">{m.onboarding_practice_description()}</p>
        </div>
        <Badge variant="outline" className="ml-auto hidden sm:inline-flex">
          {m.onboarding_practice_badge()}
        </Badge>
      </header>

      {useInlineCoach ? (
        <TutorialCoach
          stage={stage}
          rect={spotlightRect}
          compositionProgress={compositionProgress}
          isCompleting={isCompleting}
          onContinue={continueTutorial}
          presentation="inline"
        />
      ) : null}

      <div ref={boardRef} className="flex min-h-0 flex-1 flex-col">
        <GameBoardView
          game={game}
          roomCode="TUTORIAL"
          playerId={TUTORIAL_PLAYER_ID}
          players={tutorialPlayers}
          connectedPlayers={tutorialPlayers.length}
          turnState={{
            canDrawDeck: stage === "draw" && !game.turn.hasDrawn,
            canDrawDiscard: false,
            canDiscard: canCompose && game.turn.hasDrawn,
            isMyTurn: true,
            turnPlayerName: tutorialPlayers[0]?.name ?? "Alex",
          }}
          topDiscardCard={game.discardPile[0] ?? null}
          onDrawFromDeck={drawFromDeck}
          onDrawFromDiscard={() => undefined}
          guidance={{
            stage:
              stage === "orientation"
                ? "orientation"
                : stage === "draw"
                  ? "draw"
                  : stage === "compose"
                    ? "compose"
                    : "discard",
            onDrawDragStateChange: setIsDrawingCard,
            onDrawSettled: () => setStage((current) => (current === "draw" ? "compose" : current)),
            onDraftStateChange: ({ draftCompositions, isDraggingCard }) => {
              const isComplete = isTutorialDraftComplete(draftCompositions);
              setCompositionProgress(tutorialDraftProgress(draftCompositions));
              setStage((current) => {
                if (current === "compose" && isComplete && !isDraggingCard) return "discard";
                if (current === "discard" && !isComplete) return "compose";
                return current;
              });
            },
          }}
          onDiscardCard={async () => ({
            action: "discard_card",
            playerId: TUTORIAL_PLAYER_ID,
            ok: false,
          })}
          onPlayTable={async () => ({
            action: "play_table",
            playerId: TUTORIAL_PLAYER_ID,
            ok: false,
          })}
          onPlayTableAndDiscard={playTableAndDiscard}
          onSendEmote={() => undefined}
          draftSyncMode="disabled"
        />
      </div>

      <SpotlightBackdrop rects={spotlightRects} />
      {stage === "orientation" ? <OrientationLabels rects={spotlightRects} /> : null}
      {!useInlineCoach ? (
        <TutorialCoach
          stage={stage}
          rect={spotlightRect}
          compositionProgress={compositionProgress}
          isCompleting={isCompleting}
          onContinue={continueTutorial}
          presentation="overlay"
        />
      ) : null}
    </motion.section>
  );
}

export function GameOnboardingProvider({
  children,
  completedVersion,
  requiredVersion,
}: {
  children: ReactNode;
  completedVersion: number | null;
  requiredVersion: number | null;
}) {
  const [completedInSession, setCompletedInSession] = useState(false);
  const [replayRequested, setReplayRequested] = useState(false);
  const [isCompleting, setIsCompleting] = useState(false);
  const mustComplete =
    completedVersion !== null &&
    requiredVersion !== null &&
    completedVersion < requiredVersion &&
    !completedInSession;
  const open = mustComplete || replayRequested;

  function startTutorial() {
    setIsCompleting(false);
    setReplayRequested(true);
  }

  async function completeTutorial() {
    if (!mustComplete) {
      setReplayRequested(false);
      return;
    }
    if (isCompleting) return;
    setIsCompleting(true);
    try {
      await authClient.completeOnboarding();
      setCompletedInSession(true);
      setReplayRequested(false);
      setIsCompleting(false);
    } catch {
      setIsCompleting(false);
      toast.error(m.onboarding_completion_error());
    }
  }

  return (
    <GameOnboardingContext.Provider value={{ startTutorial }}>
      <div className="contents" inert={open || undefined} aria-hidden={open || undefined}>
        {children}
      </div>
      <AnimatePresence>
        {open ? <TutorialGame isCompleting={isCompleting} onComplete={completeTutorial} /> : null}
      </AnimatePresence>
    </GameOnboardingContext.Provider>
  );
}

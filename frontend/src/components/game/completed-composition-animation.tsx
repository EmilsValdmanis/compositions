import { useEffectEvent, useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { type CardSnapshot, type GameSnapshot } from "#/lib/game-protocol";
import { GameCard } from "#/components/game/game-card";
import {
  type CompletedCompositionCollection,
  inferCompletedCompositionCollection,
} from "#/components/game/card-transfer-state";
import { shouldReduceMotion } from "#/lib/reduced-motion";

type Bounds = {
  height: number;
  left: number;
  top: number;
  width: number;
};

type CachedComposition = {
  cards: Array<{ bounds: Bounds; card: CardSnapshot }>;
};

type CollectionSequence = CompletedCompositionCollection & {
  cards: CachedComposition["cards"];
  discardSource: Bounds;
  id: number;
  game: GameSnapshot;
  stage: "collect" | "discard";
  target: Bounds;
};

const COMPLETED_COLLECTION_DURATION_MS = 520;
const COMPLETED_DISCARD_DURATION_MS = 360;

function boundsFor(element: Element): Bounds {
  const bounds = element.getBoundingClientRect();
  return {
    height: bounds.height,
    left: bounds.left,
    top: bounds.top,
    width: bounds.width,
  };
}

function cardFromElement(element: HTMLElement): CardSnapshot {
  if (element.dataset.cardJoker === "true") {
    return { isJoker: true };
  }

  return {
    rank: Number(element.dataset.cardRank),
    suit: Number(element.dataset.cardSuit),
  };
}

function captureCompletedCompositions(board: HTMLElement) {
  return Array.from(board.querySelectorAll<HTMLElement>('[data-completed-composition="true"]')).map(
    (composition): CachedComposition => {
      const wrappers = Array.from(
        composition.querySelectorAll<HTMLElement>("[data-composition-card-wrap]"),
      );
      return {
        cards: wrappers.flatMap((wrapper) => {
          const card = wrapper.querySelector<HTMLElement>("[data-game-card]");
          return card ? [{ bounds: boundsFor(card), card: cardFromElement(wrapper) }] : [];
        }),
      };
    },
  );
}

function findPlayerAnchor(board: HTMLElement, playerId: string) {
  return Array.from(board.querySelectorAll<HTMLElement>("[data-card-motion-player]")).find(
    (element) => element.dataset.cardMotionPlayer === playerId,
  );
}

function findDiscardCard(board: HTMLElement) {
  return board.querySelector<HTMLElement>('[data-card-motion-source="discard"] [data-game-card]');
}

function targetCardBounds(target: Bounds, cardIndex: number): Bounds {
  const stackOffset = Math.min(cardIndex, 5) * 0.75;
  return {
    ...target,
    left: target.left + stackOffset,
    top: target.top - stackOffset,
  };
}

function cardBoundsAtAnchor(anchor: Bounds, card: Bounds): Bounds {
  return {
    ...card,
    left: anchor.left + (anchor.width - card.width) / 2,
    top: anchor.top + (anchor.height - card.height) / 2,
  };
}

export function buildCompletedCardCollectionKeyframes(source: Bounds, target: Bounds): Keyframe[] {
  const translateX = target.left + (target.width - source.width) / 2 - source.left;
  const translateY = target.top + (target.height - source.height) / 2 - source.top;
  const targetScale = Math.min(target.width / source.width, target.height / source.height);

  return [
    { opacity: 1, transform: "translate3d(0, 0, 0) scale(1) rotate(0deg)" },
    {
      offset: 0.22,
      opacity: 1,
      transform: "translate3d(0, -16px, 0) scale(1.025) rotate(-1deg)",
    },
    {
      opacity: 1,
      transform: `translate3d(${translateX}px, ${translateY}px, 0) scale(${targetScale}) rotate(1deg)`,
    },
  ];
}

function FlyingCard({
  bounds,
  card,
  cardRef,
  layer,
}: {
  bounds: Bounds;
  card: CardSnapshot;
  cardRef?: (element: HTMLDivElement | null) => void;
  layer: number;
}) {
  return (
    <div
      ref={cardRef}
      className="pointer-events-none fixed"
      style={{
        height: bounds.height,
        left: bounds.left,
        top: bounds.top,
        width: bounds.width,
        zIndex: layer,
      }}
      aria-hidden="true"
    >
      <GameCard card={card} className="shadow-xl ring-1 ring-foreground/10" />
    </div>
  );
}

export function CompletedCompositionAnimation({
  boardRef,
  game,
  viewerPlayerId,
  onSettled,
}: {
  boardRef: RefObject<HTMLDivElement | null>;
  game: GameSnapshot | null;
  viewerPlayerId: string;
  onSettled?: (game: GameSnapshot) => void;
}) {
  const previousGameRef = useRef<GameSnapshot | null>(null);
  const completedCompositionsRef = useRef<CachedComposition[]>([]);
  const nextSequenceIdRef = useRef(0);
  const collectedCardRefs = useRef<Array<HTMLDivElement | null>>([]);
  const discardCardRef = useRef<HTMLDivElement | null>(null);
  const [sequence, setSequence] = useState<CollectionSequence | null>(null);

  const notifySettled = useEffectEvent((completedGame: GameSnapshot) => onSettled?.(completedGame));

  useLayoutEffect(() => {
    const board = boardRef.current;
    const previous = previousGameRef.current;
    previousGameRef.current = game;
    const cachedCards = completedCompositionsRef.current.flatMap(
      (composition) => composition.cards,
    );
    completedCompositionsRef.current = board ? captureCompletedCompositions(board) : [];

    if (!previous || !game) return;

    const collection = inferCompletedCompositionCollection(previous, game);
    if (!collection) return;

    const discardTarget = board && findDiscardCard(board);
    if (
      !board ||
      !discardTarget ||
      !discardTarget.animate ||
      cachedCards.length !== collection.collectedCards.length ||
      sequence
    ) {
      notifySettled(game);
      return;
    }

    const target = boundsFor(discardTarget);
    if (
      !target.width ||
      !target.height ||
      cachedCards.some(({ bounds }) => !bounds.width || !bounds.height)
    ) {
      notifySettled(game);
      return;
    }
    const playerAnchor = findPlayerAnchor(board, collection.actorPlayerId);
    const discardSource =
      collection.actorPlayerId === viewerPlayerId || !playerAnchor
        ? target
        : cardBoundsAtAnchor(boundsFor(playerAnchor), target);
    const id = nextSequenceIdRef.current;
    nextSequenceIdRef.current += 1;
    setSequence({
      ...collection,
      cards: cachedCards,
      discardSource,
      game,
      id,
      stage: "collect",
      target,
    });
  }, [boardRef, game, sequence, viewerPlayerId]);

  useLayoutEffect(() => {
    if (!sequence) return;

    const reducedMotion = shouldReduceMotion();
    if (sequence.stage === "collect") {
      const animations = collectedCardRefs.current.flatMap((element, index) => {
        if (!element) return [];
        element.style.willChange = "transform, opacity";
        const animation = element.animate(
          reducedMotion
            ? [{ opacity: 1 }, { opacity: 0 }]
            : buildCompletedCardCollectionKeyframes(
                sequence.cards[index].bounds,
                targetCardBounds(sequence.target, index),
              ),
          {
            duration: reducedMotion ? 120 : COMPLETED_COLLECTION_DURATION_MS,
            easing: reducedMotion
              ? "cubic-bezier(0.23, 1, 0.32, 1)"
              : "cubic-bezier(0.77, 0, 0.175, 1)",
            fill: "forwards",
          },
        );
        return [{ element, animation }];
      });

      let active = true;
      void Promise.allSettled(animations.map(({ animation }) => animation.finished)).then(() => {
        if (active) {
          setSequence((current) =>
            current?.id === sequence.id ? { ...current, stage: "discard" } : current,
          );
        }
      });
      return () => {
        active = false;
        for (const { animation, element } of animations) {
          animation.cancel();
          element.style.willChange = "";
        }
      };
    }

    const finishSequence = () => {
      setSequence((current) => (current?.id === sequence.id ? null : current));
      notifySettled(sequence.game);
    };

    const element = discardCardRef.current;
    if (!element) {
      finishSequence();
      return;
    }
    element.style.willChange = "transform, opacity";
    const translateX =
      sequence.target.left +
      (sequence.target.width - sequence.discardSource.width) / 2 -
      sequence.discardSource.left;
    const translateY =
      sequence.target.top +
      (sequence.target.height - sequence.discardSource.height) / 2 -
      sequence.discardSource.top;
    const startsAtTarget = sequence.actorPlayerId === viewerPlayerId;
    const animation = element.animate(
      reducedMotion || startsAtTarget
        ? [
            { opacity: 0, transform: "scale(0.96)" },
            { opacity: 1, transform: "scale(1)" },
          ]
        : [
            {
              opacity: 0.35,
              transform: "translate3d(0, 0, 0) scale(0.34) rotate(-2deg)",
            },
            {
              opacity: 1,
              transform: `translate3d(${translateX}px, ${translateY}px, 0) scale(1) rotate(0deg)`,
            },
          ],
      {
        duration: reducedMotion ? 120 : COMPLETED_DISCARD_DURATION_MS,
        easing: "cubic-bezier(0.23, 1, 0.32, 1)",
        fill: "forwards",
      },
    );
    animation.onfinish = finishSequence;
    animation.oncancel = finishSequence;
    return () => {
      element.style.willChange = "";
      animation.onfinish = null;
      animation.oncancel = null;
      animation.cancel();
    };
  }, [sequence, viewerPlayerId]);

  if (!sequence || typeof document === "undefined") return null;

  const collectedAtTarget = sequence.stage === "discard";
  const discardBounds =
    sequence.actorPlayerId === viewerPlayerId ? sequence.target : sequence.discardSource;

  return createPortal(
    <>
      {sequence.previousTopDiscard ? (
        <FlyingCard bounds={sequence.target} card={sequence.previousTopDiscard} layer={60} />
      ) : (
        <div
          className="pointer-events-none fixed rounded-xl border border-dashed border-border bg-background"
          style={{ ...sequence.target, zIndex: 60 }}
          aria-hidden="true"
        />
      )}
      {sequence.cards.map(({ bounds, card }, index) => (
        <FlyingCard
          key={`${sequence.id}-collected-${index}`}
          bounds={collectedAtTarget ? targetCardBounds(sequence.target, index) : bounds}
          card={card}
          cardRef={
            collectedAtTarget
              ? undefined
              : (element) => {
                  collectedCardRefs.current[index] = element;
                }
          }
          layer={61 + index}
        />
      ))}
      {sequence.stage === "discard" ? (
        <FlyingCard
          bounds={discardBounds}
          card={sequence.discardCard}
          cardRef={(element) => {
            discardCardRef.current = element;
          }}
          layer={80}
        />
      ) : null}
    </>,
    document.body,
  );
}

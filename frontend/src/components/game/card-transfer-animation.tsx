import { useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { type GameSnapshot } from "#/components/game-websocket-provider";
import { GameCard } from "#/components/game/game-card";
import {
  type CardTransfer,
  inferCardTransfer,
  inferCompletedCompositionCollection,
} from "#/components/game/card-transfer-state";
import { shouldReduceMotion } from "#/lib/reduced-motion";

type Flight = CardTransfer & {
  id: number;
  startLeft: number;
  startTop: number;
  translateX: number;
  translateY: number;
};

export const CARD_TRANSFER_DURATION_MS = 420;
export const CARD_TRANSFER_PLAYER_SCALE = 0.34;

export function buildCardTransferKeyframes(
  flight: Pick<Flight, "source" | "target" | "translateX" | "translateY">,
): Keyframe[] {
  const movingToPlayer = flight.target === "player";
  const startScale = flight.source === "player" ? CARD_TRANSFER_PLAYER_SCALE : 1;
  const endScale = movingToPlayer ? CARD_TRANSFER_PLAYER_SCALE : 1;
  const nearTargetScale = movingToPlayer ? 0.46 : 0.94;

  return [
    {
      opacity: flight.source === "player" ? 0.35 : 1,
      transform: `translate3d(0, 0, 0) scale(${startScale}) rotate(-2deg)`,
    },
    {
      offset: 0.82,
      opacity: 1,
      transform: `translate3d(${flight.translateX}px, ${flight.translateY}px, 0) scale(${nearTargetScale}) rotate(2deg)`,
    },
    {
      opacity: movingToPlayer ? 0.15 : 1,
      transform: `translate3d(${flight.translateX}px, ${flight.translateY}px, 0) scale(${endScale}) rotate(0deg)`,
    },
  ];
}

function anchorCenter(element: Element) {
  const bounds = element.getBoundingClientRect();
  return {
    x: bounds.left + bounds.width / 2,
    y: bounds.top + bounds.height / 2,
  };
}

function findPlayerAnchor(board: HTMLElement, playerId: string) {
  return Array.from(board.querySelectorAll<HTMLElement>("[data-card-motion-player]")).find(
    (element) => element.dataset.cardMotionPlayer === playerId,
  );
}

function findPileAnchor(board: HTMLElement, pile: "deck" | "discard") {
  return board.querySelector<HTMLElement>(`[data-card-motion-source="${pile}"]`);
}

export function CardTransferAnimation({
  boardRef,
  game,
  viewerPlayerId,
}: {
  boardRef: RefObject<HTMLDivElement | null>;
  game: GameSnapshot | null;
  viewerPlayerId: string;
}) {
  const previousGameRef = useRef<GameSnapshot | null>(null);
  const nextFlightIdRef = useRef(0);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [flight, setFlight] = useState<Flight | null>(null);

  useLayoutEffect(() => {
    const previous = previousGameRef.current;
    previousGameRef.current = game;

    if (!previous || !game || !boardRef.current) return;

    const transfer = inferCardTransfer(previous, game, viewerPlayerId);
    if (!transfer) return;
    if (transfer.source === "player" && inferCompletedCompositionCollection(previous, game)) {
      return;
    }

    const playerAnchor = findPlayerAnchor(boardRef.current, transfer.actorPlayerId);
    const pileAnchor = findPileAnchor(
      boardRef.current,
      transfer.source === "deck" ? "deck" : "discard",
    );
    const sourceAnchor = transfer.source === "player" ? playerAnchor : pileAnchor;
    const targetAnchor = transfer.target === "player" ? playerAnchor : pileAnchor;

    if (!sourceAnchor || !targetAnchor) return;

    const start = anchorCenter(sourceAnchor);
    const target = anchorCenter(targetAnchor);
    const id = nextFlightIdRef.current;
    nextFlightIdRef.current += 1;
    setFlight({
      ...transfer,
      id,
      startLeft: start.x - 32,
      startTop: start.y - 48,
      translateX: target.x - start.x,
      translateY: target.y - start.y,
    });
  }, [boardRef, game, viewerPlayerId]);

  useLayoutEffect(() => {
    const element = cardRef.current;
    if (!element || !flight) return;

    const reducedMotion = shouldReduceMotion();
    element.style.willChange = "transform, opacity";
    const animation = element.animate(
      reducedMotion
        ? [{ opacity: 0 }, { opacity: 1 }, { opacity: 0 }]
        : buildCardTransferKeyframes(flight),
      {
        duration: reducedMotion ? 160 : CARD_TRANSFER_DURATION_MS,
        easing: reducedMotion
          ? "cubic-bezier(0.23, 1, 0.32, 1)"
          : "cubic-bezier(0.77, 0, 0.175, 1)",
        fill: "forwards",
      },
    );

    animation.onfinish = () => {
      element.style.willChange = "";
      setFlight((current) => (current?.id === flight.id ? null : current));
    };
    return () => {
      element.style.willChange = "";
      animation.cancel();
    };
  }, [flight]);

  if (!flight || typeof document === "undefined") return null;

  return createPortal(
    <div
      ref={cardRef}
      className="pointer-events-none fixed z-50"
      style={{ left: flight.startLeft, top: flight.startTop }}
      aria-hidden="true"
      data-card-transfer={flight.source === "player" ? "discard-place" : `${flight.source}-draw`}
    >
      <GameCard
        card={flight.card}
        faceDown={flight.faceDown}
        className="shadow-xl ring-1 ring-foreground/10"
      />
    </div>,
    document.body,
  );
}

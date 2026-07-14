import { useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { type GameSnapshot } from "#/components/game-websocket-provider";
import { GameCard } from "#/components/game/game-card";
import { type CardTransfer, inferCardTransfer } from "#/components/game/card-transfer-state";

type Flight = CardTransfer & {
  id: number;
  startLeft: number;
  startTop: number;
  translateX: number;
  translateY: number;
};

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

    const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    const movingToPlayer = flight.target === "player";
    const animation = element.animate(
      reducedMotion
        ? [{ opacity: 0 }, { opacity: 1 }, { opacity: 0 }]
        : [
            {
              opacity: movingToPlayer ? 1 : 0.2,
              transform: `translate3d(0, 0, 0) scale(${movingToPlayer ? 1 : 0.72}) rotate(-2deg)`,
            },
            {
              offset: 0.82,
              opacity: 1,
              transform: `translate3d(${flight.translateX}px, ${flight.translateY}px, 0) scale(${movingToPlayer ? 0.76 : 0.96}) rotate(2deg)`,
            },
            {
              opacity: movingToPlayer ? 0.15 : 1,
              transform: `translate3d(${flight.translateX}px, ${flight.translateY}px, 0) scale(${movingToPlayer ? 0.72 : 1}) rotate(0deg)`,
            },
          ],
      {
        duration: reducedMotion ? 160 : 280,
        easing: reducedMotion
          ? "cubic-bezier(0.23, 1, 0.32, 1)"
          : "cubic-bezier(0.77, 0, 0.175, 1)",
        fill: "forwards",
      },
    );

    animation.onfinish = () => setFlight((current) => (current?.id === flight.id ? null : current));
    return () => animation.cancel();
  }, [flight]);

  if (!flight || typeof document === "undefined") return null;

  return createPortal(
    <div
      ref={cardRef}
      className="pointer-events-none fixed z-50 will-change-transform"
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

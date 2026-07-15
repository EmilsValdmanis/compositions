import { type CardSnapshot } from "#/components/game-websocket-provider";
import { DiscardDropZone } from "#/components/game/discard-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { FACE_DOWN_CARD } from "#/components/game/game-board-view-state";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Card, CardContent } from "#/components/ui/card";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

const PILE_CARD_GAP_PX = 0.22;
const HORIZONTAL_CARD_CLASS_NAME =
  "h-14! w-20! rounded-xl! bg-card! p-1.5! xl:h-16! xl:w-24! xl:p-2! [@media(max-height:600px)]:h-12! [@media(max-height:600px)]:w-17!";

function pileLayerOffset(index: number, pileSize: number) {
  if (pileSize <= 1) return 0;
  return (pileSize - 1 - index) * PILE_CARD_GAP_PX;
}

function pileCardTransform(offset: number) {
  return `translate(-50%, calc(-50% + ${offset}px))`;
}

function PileCardLayer({
  index,
  pileSize,
  faceDown,
}: {
  index: number;
  pileSize: number;
  faceDown: boolean;
}) {
  const offset = pileLayerOffset(index, pileSize);

  return (
    <div
      className={cn(
        "absolute left-1/2 top-1/2 h-14 w-20 rounded-xl border bg-card shadow-[0_1px_0_hsl(var(--border))] xl:h-16 xl:w-24 [@media(max-height:600px)]:h-12 [@media(max-height:600px)]:w-17",
        faceDown ? "border-border" : "border-foreground/15",
      )}
      style={{
        zIndex: index,
        transform: pileCardTransform(offset),
      }}
      aria-hidden="true"
    >
      {faceDown ? (
        <span className="absolute inset-1 rounded-lg border border-foreground/10 bg-muted" />
      ) : null}
    </div>
  );
}

function TableCardPile({
  count,
  card,
  faceDown,
  dragSource,
  countOnCard,
  motionSource,
}: {
  count: number;
  card: CardSnapshot;
  faceDown?: boolean;
  dragSource: {
    id: string;
    disabled: boolean;
    data: { drawSource: "deck" | "discard" };
  };
  countOnCard?: boolean;
  motionSource: "deck" | "discard";
}) {
  const pileSize = Math.max(1, count);
  const topOffset = pileLayerOffset(pileSize - 1, pileSize);
  const accessibleLabel = `${motionSource === "deck" ? m.draw() : m.discard()}: ${m.cards_count({ count })}`;

  return (
    <div
      className="relative h-20 min-w-0 xl:h-24 [@media(max-height:600px)]:h-full"
      data-card-motion-source={motionSource}
      role="group"
      aria-label={accessibleLabel}
    >
      <div className="pointer-events-none absolute bottom-1.5 left-1/2 h-1.5 w-20 -translate-x-1/2 rounded-full bg-black/35 blur-md xl:w-24 [@media(max-height:600px)]:w-17" />

      {Array.from({ length: Math.max(0, count - 1) }, (_, index) => (
        <PileCardLayer key={index} index={index} pileSize={pileSize} faceDown={Boolean(faceDown)} />
      ))}

      {count > 0 ? (
        <div
          className="absolute left-1/2 top-1/2"
          style={{
            zIndex: pileSize,
            transform: pileCardTransform(topOffset),
          }}
        >
          <GameCard
            card={card}
            faceDown={faceDown}
            dragSource={dragSource}
            className={HORIZONTAL_CARD_CLASS_NAME}
          />
          {countOnCard ? (
            <span className="pointer-events-none absolute inset-0 grid place-items-center">
              <span className="grid min-w-7 place-items-center rounded-full bg-background/90 px-1.5 py-1 font-heading text-xs font-semibold text-foreground shadow-sm ring-1 ring-foreground/10 backdrop-blur-sm">
                <AnimatedNumber value={count} />
              </span>
            </span>
          ) : null}
        </div>
      ) : (
        <div
          className="absolute left-1/2 top-1/2 grid h-14 w-20 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-xl border border-dashed border-border/70 xl:h-16 xl:w-24 [@media(max-height:600px)]:h-12 [@media(max-height:600px)]:w-17"
          aria-hidden="true"
        >
          {countOnCard ? (
            <AnimatedNumber className="font-heading text-xs text-muted-foreground" value={0} />
          ) : null}
        </div>
      )}
    </div>
  );
}

export function GameBoardPiles({
  drawPileCount,
  discardPileCount,
  topDiscardCard,
  canDrawDeck,
  canDrawDiscard,
  canDiscard,
}: {
  drawPileCount: number;
  discardPileCount: number;
  topDiscardCard: CardSnapshot | null;
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
}) {
  return (
    <Card
      size="sm"
      className="h-fit [--card-spacing:--spacing(2)] xl:[--card-spacing:--spacing(4)] [@media(max-height:600px)]:h-full [@media(max-height:600px)]:py-0!"
    >
      <CardContent className="grid min-h-0 grid-cols-2 items-center gap-1 xl:gap-2 [@media(max-height:600px)]:h-full">
        <TableCardPile
          count={drawPileCount}
          card={FACE_DOWN_CARD}
          faceDown
          dragSource={{
            id: "draw-pile",
            disabled: !canDrawDeck,
            data: { drawSource: "deck" },
          }}
          countOnCard
          motionSource="deck"
        />

        <DiscardDropZone disabled={!canDiscard}>
          <TableCardPile
            count={discardPileCount}
            card={topDiscardCard ?? FACE_DOWN_CARD}
            dragSource={{
              id: "discard-draw",
              disabled: !canDrawDiscard,
              data: { drawSource: "discard" },
            }}
            motionSource="discard"
          />
        </DiscardDropZone>
      </CardContent>
    </Card>
  );
}

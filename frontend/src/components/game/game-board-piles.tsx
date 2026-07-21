import { type CardSnapshot } from "#/components/game-websocket-provider";
import { DiscardDropZone } from "#/components/game/discard-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { FACE_DOWN_CARD } from "#/components/game/game-board-view-state";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Card, CardContent } from "#/components/ui/card";
import { m } from "#/paraglide/messages.js";
import { useIsMobile } from "#/hooks/use-mobile";

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
  const accessibleLabel = `${motionSource === "deck" ? m.draw() : m.discard()}: ${m.cards_count({ count })}`;

  return (
    <div
      className="grid min-w-0 place-items-center"
      data-card-motion-source={motionSource}
      role="group"
      aria-label={accessibleLabel}
    >
      {count > 0 ? (
        <div className="relative">
          <GameCard card={card} faceDown={faceDown} dragSource={dragSource} />
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
          className="grid h-20 w-14 place-items-center rounded-2xl border border-dashed border-border/70 xl:h-24 xl:w-16"
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
  const isMobile = useIsMobile();
  return (
    <Card size={isMobile ? "sm" : "default"}>
      <CardContent className="grid grid-cols-2 items-center gap-2">
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

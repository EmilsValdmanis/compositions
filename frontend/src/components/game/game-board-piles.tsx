import { type CardSnapshot } from "#/components/game-websocket-provider";
import { DiscardDropZone } from "#/components/game/discard-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { FACE_DOWN_CARD } from "#/components/game/game-board-view-state";
import { Card, CardContent } from "#/components/ui/card";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Text } from "#/components/typography";

export function GameBoardPiles({
  drawPileCount,
  topDiscardCard,
  canDrawDeck,
  canDrawDiscard,
  canDiscard,
}: {
  drawPileCount: number;
  topDiscardCard: CardSnapshot | null;
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
}) {
  return (
    <Card size="sm" className="h-fit">
      <CardContent className="grid gap-3 grid-cols-2">
        <div className="rounded-3xl border flex flex-col border-border/70 bg-muted/20 p-3">
          <Text
            as="div"
            variant="eyebrow-compact"
            className="mb-2 flex items-center justify-between gap-3"
          >
            <span>Draw</span>
            <span>
              <AnimatedNumber value={drawPileCount} /> cards
            </span>
          </Text>
          <div
            className="flex items-center justify-center grow flex-col"
            data-card-motion-source="deck"
          >
            <GameCard
              card={FACE_DOWN_CARD}
              faceDown
              dragSource={{
                id: "draw-pile",
                disabled: !canDrawDeck,
                data: { drawSource: "deck" },
              }}
            />
          </div>
        </div>
        <DiscardDropZone disabled={!canDiscard}>
          <Text
            as="div"
            variant="eyebrow-compact"
            className="mb-2 flex items-center justify-between gap-3"
          >
            <span>Discard</span>
            <span>{canDrawDiscard ? "Can draw" : "Top card"}</span>
          </Text>
          <div className="flex items-center justify-center" data-card-motion-source="discard">
            {topDiscardCard ? (
              <GameCard
                card={topDiscardCard}
                dragSource={{
                  id: "discard-draw",
                  disabled: !canDrawDiscard,
                  data: { drawSource: "discard" },
                }}
                className={canDrawDiscard ? "shadow-md" : undefined}
              />
            ) : (
              <Text className="text-muted-foreground">Empty</Text>
            )}
          </div>
        </DiscardDropZone>
      </CardContent>
    </Card>
  );
}

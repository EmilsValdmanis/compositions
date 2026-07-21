import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Card, CardContent } from "#/components/ui/card";
import { Caption } from "#/components/typography";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { useIsMobile } from "#/hooks/use-mobile";

type HandStatus = {
  hasGame: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
};

type TablePlayState = {
  hasDraftedCompositions: boolean;
};

type GameBoardHandProps = {
  status: HandStatus;
  availableHandEntries: HandEntry[];
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  tablePlayState: TablePlayState;
};

export function GameBoardHand({
  status,
  availableHandEntries,
  sortableIds,
  activeDrag,
  tablePlayState,
}: GameBoardHandProps) {
  const { hasGame } = status;
  const { hasDraftedCompositions } = tablePlayState;
  const isMobile = useIsMobile();
  return (
    <Card size={isMobile ? "sm" : "default"} className="min-h-0 shrink-0 pb-0">
      <CardContent className="min-h-0 px-0">
        {hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="min-h-0 rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="min-h-0 touch-pan-x overflow-x-auto overflow-y-hidden overscroll-x-contain px-(--card-spacing) scroll-fade-x pb-1">
                  <div className="flex w-max min-w-full justify-start pb-(--card-spacing) gap-2 xl:justify-center">
                    {availableHandEntries.map((entry) => (
                      <GameCard
                        key={entry.key}
                        card={entry.card}
                        size="hand"
                        draggable={{
                          id: entry.key,
                          cardIndex: entry.sourceIndex,
                          isVirtual: entry.isVirtual,
                        }}
                        className={cn(
                          activeDrag?.type === "draw" && entry.key === activeDrag.revealedHandKey
                            ? "invisible"
                            : undefined,
                        )}
                      />
                    ))}
                  </div>
                </div>
              ) : (
                <Caption className="rounded-3xl border border-dashed border-border/70 p-6">
                  {hasDraftedCompositions ? m.all_cards_staged() : m.no_cards_in_hand()}
                </Caption>
              )}
            </GameBoardDraftDropZone>
          </SortableContext>
        ) : (
          <Caption className="rounded-3xl border border-dashed border-border/70 p-6">
            {m.waiting_game_snapshot()}
          </Caption>
        )}
      </CardContent>
    </Card>
  );
}

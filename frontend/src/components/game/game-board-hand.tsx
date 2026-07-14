import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Card, CardContent } from "#/components/ui/card";
import { Text } from "#/components/typography";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

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

  return (
    <Card className="min-h-0">
      <CardContent className="min-h-0">
        {hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="min-h-0 rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="min-h-0 overflow-x-auto overflow-y-hidden">
                  <div className="flex w-max min-w-full justify-center gap-2">
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
                <Text
                  as="div"
                  className="rounded-3xl border border-dashed border-border/70 p-6 text-muted-foreground"
                >
                  {hasDraftedCompositions ? m.all_cards_staged() : m.no_cards_in_hand()}
                </Text>
              )}
            </GameBoardDraftDropZone>
          </SortableContext>
        ) : (
          <Text
            as="div"
            className="rounded-3xl border border-dashed border-border/70 p-6 text-muted-foreground"
          >
            {m.waiting_game_snapshot()}
          </Text>
        )}
      </CardContent>
    </Card>
  );
}

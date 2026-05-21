import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { cn } from "#/lib/utils";

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
  const { hasGame, isMyTurn, turnPlayerName } = status;
  const { hasDraftedCompositions } = tablePlayState;

  return (
    <Card className="min-h-0 overflow-hidden">
      <CardHeader>
        <CardTitle>Hand</CardTitle>
        <CardDescription>{isMyTurn ? "Your turn" : `${turnPlayerName} is up`}</CardDescription>
        <Badge variant="outline" className="w-fit">
          {availableHandEntries.length} cards
        </Badge>
      </CardHeader>
      <CardContent className="min-h-0">
        {hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="min-h-0 rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="min-h-0 overflow-x-auto overflow-y-hidden pb-2">
                  <div className="flex w-max min-w-full justify-center gap-2">
                    {availableHandEntries.map((entry) => (
                      <GameCard
                        key={entry.key}
                        card={entry.card}
                        size="hand"
                        draggable={{ id: entry.key, cardIndex: entry.sourceIndex }}
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
                <div className="rounded-3xl border border-dashed border-border/70 p-6 text-sm text-muted-foreground">
                  {hasDraftedCompositions
                    ? "All of your cards are staged in compositions. Drag one back here to return it to hand."
                    : "No cards in hand."}
                </div>
              )}
            </GameBoardDraftDropZone>
          </SortableContext>
        ) : (
          <div className="rounded-3xl border border-dashed border-border/70 p-6 text-sm text-muted-foreground">
            Waiting for the first game snapshot.
          </div>
        )}
      </CardContent>
    </Card>
  );
}

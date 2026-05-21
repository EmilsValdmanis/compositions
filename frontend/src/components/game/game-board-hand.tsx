import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import { cn } from "#/lib/utils";

type HandStatus = {
  hasGame: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
};

type TablePlayState = {
  hasDraftedCompositions: boolean;
  canSubmitTablePlay: boolean;
};

type GameBoardHandProps = {
  status: HandStatus;
  availableHandEntries: HandEntry[];
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  tablePlayState: TablePlayState;
  onResetTablePlay: () => void;
  onSubmitTablePlay: () => void;
};

export function GameBoardHand({
  status,
  availableHandEntries,
  sortableIds,
  activeDrag,
  tablePlayState,
  onResetTablePlay,
  onSubmitTablePlay,
}: GameBoardHandProps) {
  const { hasGame, isMyTurn, turnPlayerName } = status;
  const { hasDraftedCompositions, canSubmitTablePlay } = tablePlayState;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Hand</CardTitle>
        <CardDescription>{isMyTurn ? "Your turn" : `${turnPlayerName} is up`}</CardDescription>
        <CardAction>
          <div className="flex items-center gap-2">
            <Badge variant="outline">{availableHandEntries.length} cards</Badge>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={onResetTablePlay}
              disabled={!hasDraftedCompositions}
            >
              Reset
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={onSubmitTablePlay}
              disabled={!canSubmitTablePlay}
            >
              Submit table play
            </Button>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent>
        {hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="flex min-h-36 flex-wrap justify-center gap-2 pb-2">
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

import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  type DraftedCompositionView,
  type TableCompositionView,
  draftCompositionDropId,
} from "#/components/game/game-board-view-state";
import { type GameSnapshot } from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";

export function GameBoardTable({
  game,
  tableCompositions,
  newCompositions,
  canCompose,
  hasDraftedCompositions,
  canSubmitTablePlay,
  onResetTablePlay,
  onSubmitTablePlay,
}: {
  game: GameSnapshot | null;
  tableCompositions: TableCompositionView[];
  newCompositions: DraftedCompositionView[];
  canCompose: boolean;
  hasDraftedCompositions: boolean;
  canSubmitTablePlay: boolean;
  onResetTablePlay: () => void;
  onSubmitTablePlay: () => void;
}) {
  const tablePoints = (game?.activeCompositions ?? []).reduce(
    (total, composition) => total + composition.points,
    0,
  );
  const hasVisibleCompositions = tableCompositions.length > 0;

  return (
    <Card className="min-h-0 overflow-y-scroll xl:flex-1">
      <CardHeader>
        <CardTitle>Table</CardTitle>
        <CardAction>
          <Badge variant="outline">{tablePoints} pts</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-4">
        <div className="min-h-0">
          {hasVisibleCompositions ? (
            <div className="flex min-h-0 flex-wrap items-start gap-3">
              {tableCompositions.map((composition) => (
                <div key={composition.key} className="w-fit shrink-0">
                  <CompositionRow
                    composition={composition.snapshot}
                    index={composition.tableIndex}
                    stagedEntries={composition.stagedEntries}
                    reclaims={composition.reclaims}
                  />
                </div>
              ))}
            </div>
          ) : (
            <div className="grid min-h-32 place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center text-sm text-muted-foreground">
              No compositions on the table.
            </div>
          )}
        </div>

        <div className="flex flex-wrap justify-center gap-3">
          {newCompositions.map((composition) => (
            <GameBoardDraftDropZone
              key={composition.id}
              id={draftCompositionDropId(composition.id)}
              className="flex w-fit flex-col shrink-0 rounded-3xl border border-dashed border-border/70 bg-muted/10 p-3"
            >
              <div className="mb-3 flex justify-between items-center gap-2">
                <Badge variant="secondary">New</Badge>
                <Badge variant="outline">{composition.entries.length} cards</Badge>
              </div>
              <SortableContext
                items={composition.entries.map((entry) => entry.key)}
                strategy={horizontalListSortingStrategy}
              >
                <div className="flex items-start gap-2">
                  {composition.entries.map((entry) => (
                    <GameCard
                      key={entry.key}
                      card={entry.card}
                      size="compact"
                      draggable={{
                        id: entry.key,
                        cardIndex: entry.sourceIndex,
                        isVirtual: entry.isVirtual,
                      }}
                    />
                  ))}
                </div>
              </SortableContext>
            </GameBoardDraftDropZone>
          ))}

          <GameBoardDraftDropZone
            id={NEW_COMPOSITION_DROP_ID}
            className="grid min-h-32 w-full max-w-80 min-w-64 shrink-0 place-items-center rounded-3xl border border-dashed border-border/70 px-4 py-6 text-center text-sm text-muted-foreground"
          >
            {canCompose
              ? "Drop cards here to start a new composition."
              : hasVisibleCompositions
                ? "Additions to table compositions are shown above."
                : "Waiting for a composition to be played."}
          </GameBoardDraftDropZone>
        </div>
      </CardContent>
      <CardFooter className="flex justify-between">
        <div className="space-y-1 text-sm">
          <div className="font-medium">New compositions</div>
          <div className="text-muted-foreground">
            {hasDraftedCompositions
              ? "Review your staged cards, then submit them to the table."
              : "Build a new composition or add cards to one already on the table."}
          </div>
        </div>
        <div className="flex items-center gap-2">
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
            size="default"
            onClick={onSubmitTablePlay}
            disabled={!canSubmitTablePlay}
            className="shadow-sm"
          >
            Submit table play
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}

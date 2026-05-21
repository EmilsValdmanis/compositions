import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  type TableCompositionView,
  draftCompositionDropId,
} from "#/components/game/game-board-view-state";
import { type GameSnapshot } from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import {
  Card,
  CardContent,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";

export function GameBoardTable({
  game,
  tableCompositions,
  canCompose,
}: {
  game: GameSnapshot | null;
  tableCompositions: TableCompositionView[];
  canCompose: boolean;
}) {
  const tablePoints = (game?.activeCompositions ?? []).reduce(
    (total, composition) => total + composition.points,
    0,
  );

  return (
    <Card className="min-h-0 xl:flex-1">
      <CardHeader>
        <CardTitle>Table</CardTitle>
        <CardDescription>
          {game?.activeCompositions.length ?? 0} compositions on the table
        </CardDescription>
        <CardAction>
          <Badge variant="outline">{tablePoints} pts</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="flex h-full flex-wrap content-start justify-center gap-3">
        {tableCompositions.length ? (
          tableCompositions.map((composition, index) =>
            composition.snapshot ? (
              <CompositionRow
                key={composition.key}
                composition={composition.snapshot}
                index={composition.tableIndex ?? index}
                stagedEntries={composition.entries}
              />
            ) : (
              <GameBoardDraftDropZone
                key={composition.key}
                id={draftCompositionDropId(composition.key)}
                className="w-fit max-w-full rounded-3xl border border-dashed border-border/70 bg-muted/10 p-3"
              >
                <div className="mb-3 flex flex-wrap items-center gap-2">
                  <Badge variant="secondary">New</Badge>
                  <Badge variant="outline">{composition.entries.length} cards</Badge>
                </div>
                <SortableContext
                  items={composition.entries.map((entry) => entry.key)}
                  strategy={horizontalListSortingStrategy}
                >
                  <div className="flex min-h-16 flex-wrap justify-center gap-2">
                    {composition.entries.map((entry) => (
                      <GameCard
                        key={entry.key}
                        card={entry.card}
                        size="compact"
                        draggable={{ id: entry.key, cardIndex: entry.sourceIndex }}
                      />
                    ))}
                  </div>
                </SortableContext>
              </GameBoardDraftDropZone>
            ),
          )
        ) : (
          <GameBoardDraftDropZone
            id={NEW_COMPOSITION_DROP_ID}
            className="grid min-h-56 w-full place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center text-sm text-muted-foreground"
          >
            {canCompose
              ? "Drop cards here to start a new composition."
              : "No compositions on the table."}
          </GameBoardDraftDropZone>
        )}
      </CardContent>
    </Card>
  );
}

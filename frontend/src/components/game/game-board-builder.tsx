import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  NEW_COMPOSITION_DROP_ID,
  type DraftedCompositionView,
  draftCompositionDropId,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";

export function GameBoardBuilder({
  compositions,
  canSubmit,
  onReset,
  onSubmit,
}: {
  compositions: DraftedCompositionView[];
  canSubmit: boolean;
  onReset: () => void;
  onSubmit: () => void;
}) {
  return (
    <Card size="sm">
      <CardContent className="flex flex-col gap-3 py-0">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="secondary">Builder</Badge>
            <Badge variant="outline">{compositions.length} compositions</Badge>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={onReset}
              disabled={compositions.length === 0}
            >
              All back in hand
            </Button>
            <Button type="button" size="sm" onClick={onSubmit} disabled={!canSubmit}>
              Submit to table
            </Button>
          </div>
        </div>
        <p className="text-xs text-muted-foreground">
          Drag cards from your hand into a composition lane, then submit and let the backend
          identify and validate each play.
        </p>
        <div className="flex flex-wrap items-start justify-center gap-2">
          <GameBoardDraftDropZone
            id={NEW_COMPOSITION_DROP_ID}
            className="grid min-h-28 min-w-64 place-items-center rounded-3xl border border-dashed border-border/70 px-4 py-3 text-center text-sm text-muted-foreground"
          >
            Drop a card here to start a new composition.
          </GameBoardDraftDropZone>
          {compositions.length ? (
            <>
              {compositions.map((composition, index) => (
                <div
                  key={composition.id}
                  className="w-fit max-w-full rounded-3xl border border-border/70 bg-muted/20 p-3"
                >
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="secondary">#{index + 1}</Badge>
                      <Badge variant="outline">{composition.entries.length} cards</Badge>
                    </div>
                  </div>
                  <GameBoardDraftDropZone
                    id={draftCompositionDropId(composition.id)}
                    className="w-fit max-w-full rounded-2xl border border-border/70 bg-background/70 p-2"
                  >
                    <SortableContext
                      items={composition.entries.map((entry) => entry.key)}
                      strategy={horizontalListSortingStrategy}
                    >
                      <div className="flex min-h-20 flex-wrap gap-2">
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
                </div>
              ))}
            </>
          ) : (
            <div className="rounded-3xl border border-dashed border-border/70 px-4 py-3 text-sm text-muted-foreground">
              No compositions staged yet.
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

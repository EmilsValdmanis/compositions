import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { type HandEntry, tableCompositionDropId } from "#/components/game/game-board-view-state";
import { type CompositionSnapshot } from "#/components/game-websocket-provider";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { formatLabel } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";

export function CompositionRow({
  composition,
  index,
  stagedEntries = [],
}: {
  composition: CompositionSnapshot;
  index: number;
  stagedEntries?: HandEntry[];
}) {
  return (
    <GameBoardDraftDropZone
      id={tableCompositionDropId(index)}
      className="inline-flex max-w-full flex-col rounded-3xl border border-border/70 bg-muted/20 p-3"
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">#{index + 1}</Badge>
          <Badge variant="outline">{formatLabel(composition.type)}</Badge>
          {composition.complete ? <Badge>Complete</Badge> : null}
        </div>
        <span className="text-xs text-muted-foreground">{composition.points} pts</span>
      </div>
      <div className="flex min-h-16 flex-wrap justify-center gap-2">
        {composition.cards.map((card, cardIndex) => (
          <GameCard key={`${index}-${cardIndex}`} card={card} size="compact" />
        ))}
        <SortableContext
          items={stagedEntries.map((entry) => entry.key)}
          strategy={horizontalListSortingStrategy}
        >
          {stagedEntries.map((entry) => (
            <GameCard
              key={entry.key}
              card={entry.card}
              size="compact"
              draggable={{ id: entry.key, cardIndex: entry.sourceIndex }}
            />
          ))}
        </SortableContext>
      </div>
    </GameBoardDraftDropZone>
  );
}

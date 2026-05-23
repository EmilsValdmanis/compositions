import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  buildHandEntries,
  canReclaimJokerWithCard,
  type DraftedCompositionView,
  type HandEntry,
  type PlannedJokerReclaim,
  type TableCompositionView,
  draftCompositionDropId,
} from "#/components/game/game-board-view-state";
import {
  type CompositionActivitySnapshot,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type TurnActivitySnapshot,
} from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { NewActivityLabel } from "#/components/game/game-view-utils";
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
import { cn } from "#/lib/utils";

function draftCardKey(card: DraftCompositionSnapshot["cards"][number]) {
  return card.isJoker ? "joker" : `${card.rank ?? "unknown"}-${card.suit ?? "unknown"}`;
}

function draftCardInstances(cards: DraftCompositionSnapshot["cards"]) {
  const duplicateCounts = new Map<string, number>();

  return cards.map((card) => {
    const baseKey = draftCardKey(card);
    const duplicateCount = duplicateCounts.get(baseKey) ?? 0;

    duplicateCounts.set(baseKey, duplicateCount + 1);

    return {
      card,
      key: `${baseKey}-${duplicateCount}`,
    };
  });
}

function draftPreviewForComposition(
  tableComposition: TableCompositionView | undefined,
  draft: DraftCompositionSnapshot,
  cards: DraftCompositionSnapshot["cards"],
) {
  const stagedEntries = buildHandEntries(cards) as HandEntry[];

  if (!tableComposition) {
    return {
      stagedEntries,
      reclaims: [] as PlannedJokerReclaim[],
      insertIndex: draft.insertIndex,
      cardInsertIndices: draft.cardInsertIndices,
    };
  }

  const insertIndex = draft.insertIndex ?? tableComposition.snapshot.cards.length;
  const reclaims = stagedEntries.flatMap((entry) => {
    const jokerIndex = draft.reclaimTargets?.[entry.key];
    return typeof jokerIndex === "number" &&
      canReclaimJokerWithCard(tableComposition.snapshot, jokerIndex, entry.card)
      ? [{ jokerIndex, replacementEntry: entry } satisfies PlannedJokerReclaim]
      : [];
  });
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  return {
    stagedEntries: stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key)),
    reclaims,
    insertIndex,
    cardInsertIndices: draft.cardInsertIndices,
  };
}

export function GameBoardTable({
  game,
  tableCompositions,
  newCompositions,
  players,
  turnActivity,
  canCompose,
  hasDraftedCompositions,
  canSubmitTablePlay,
  onResetTablePlay,
  onSubmitTablePlay,
}: {
  game: GameSnapshot | null;
  tableCompositions: TableCompositionView[];
  newCompositions: DraftedCompositionView[];
  players: PlayerSnapshot[];
  turnActivity?: TurnActivitySnapshot;
  canCompose: boolean;
  hasDraftedCompositions: boolean;
  canSubmitTablePlay: boolean;
  onResetTablePlay: () => void;
  onSubmitTablePlay: () => void;
}) {
  const { active } = useDndContext();
  const { setNodeRef, isOver: isOverNewCompositionBoard } = useDroppable({
    id: NEW_COMPOSITION_DROP_ID,
    disabled: !canCompose,
  });
  const tablePoints = (game?.activeCompositions ?? []).reduce(
    (total: number, composition: GameSnapshot["activeCompositions"][number]) =>
      total + composition.points,
    0,
  );
  const hasVisibleCompositions = tableCompositions.length > 0;
  const activityByIndex = new Map<number, CompositionActivitySnapshot>(
    (turnActivity?.compositionActivities ?? []).map((activity: CompositionActivitySnapshot) => [
      activity.tableIndex,
      activity,
    ]),
  );
  const stagedDrafts = turnActivity?.draftCompositions ?? [];
  const tableCompositionsByIndex = new Map(
    tableCompositions.map((composition) => [composition.tableIndex, composition]),
  );
  const spectatorDraftsByTableIndex = new Map<
    number,
    {
      stagedEntries: HandEntry[];
      reclaims: PlannedJokerReclaim[];
      insertIndex?: number;
      cardInsertIndices?: Record<string, number>;
      playerId?: string;
    }
  >();
  const stagedNewDrafts = stagedDrafts.filter(
    (composition) => composition.tableIndex === undefined,
  );
  const isDraggingHandCard =
    active !== null &&
    active.data.current?.drawSource === undefined &&
    typeof active.id === "string";

  for (const draft of stagedDrafts) {
    if (draft.tableIndex === undefined) {
      continue;
    }

    spectatorDraftsByTableIndex.set(draft.tableIndex, {
      ...draftPreviewForComposition(
        tableCompositionsByIndex.get(draft.tableIndex),
        draft,
        draft.cards,
      ),
      playerId: turnActivity?.playerId,
    });
  }

  return (
    <Card className="min-h-0 overflow-visible xl:flex-1">
      <CardHeader>
        <CardTitle>Table</CardTitle>
        <CardAction>
          <Badge variant="outline">{tablePoints} pts</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="min-h-0 flex-1 overflow-visible">
        <div
          ref={setNodeRef}
          className={[
            "flex min-h-full flex-col justify-center gap-6 rounded-3xl border border-dashed border-border/70 px-3 py-3 transition-colors",
            canCompose && isDraggingHandCard ? "border-border/70" : null,
            canCompose && isDraggingHandCard && isOverNewCompositionBoard
              ? "border-primary/70 bg-primary/5"
              : null,
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <div className="min-h-0 overflow-visible">
            {hasVisibleCompositions ? (
              <div className="flex min-h-0 flex-wrap items-center justify-center gap-4 overflow-visible p-1">
                {tableCompositions.map((composition) => (
                  <div key={composition.key} className="w-fit shrink-0 overflow-visible p-1">
                    {(() => {
                      const spectatorDraft = spectatorDraftsByTableIndex.get(
                        composition.tableIndex,
                      );
                      const stagedEntries =
                        composition.stagedEntries.length > 0
                          ? composition.stagedEntries
                          : (spectatorDraft?.stagedEntries ?? []);
                      const reclaims =
                        composition.reclaims.length > 0
                          ? composition.reclaims
                          : (spectatorDraft?.reclaims ?? []);
                      const insertIndex =
                        composition.stagedEntries.length > 0
                          ? composition.insertIndex
                          : spectatorDraft?.insertIndex;
                      const cardInsertIndices =
                        composition.stagedEntries.length > 0
                          ? composition.cardInsertIndices
                          : spectatorDraft?.cardInsertIndices;

                      return (
                        <CompositionRow
                          composition={composition.snapshot}
                          index={composition.tableIndex}
                          stagedEntries={stagedEntries}
                          reclaims={reclaims}
                          insertIndex={insertIndex}
                          cardInsertIndices={cardInsertIndices}
                          players={players}
                          stagedEntryPlayerId={spectatorDraft?.playerId}
                          stagedEntriesInteractive={composition.stagedEntries.length > 0}
                          dropTargetsEnabled={canCompose}
                          activity={activityByIndex.get(composition.tableIndex)}
                        />
                      );
                    })()}
                  </div>
                ))}
              </div>
            ) : (
              <div className="mx-auto grid min-h-32 w-fit max-w-full place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center text-sm text-muted-foreground">
                No compositions on the table.
              </div>
            )}
          </div>

          <div className="flex flex-wrap items-center justify-center gap-3">
            {stagedNewDrafts.map((composition: DraftCompositionSnapshot, index: number) => (
              <div
                key={`turn-draft-${composition.tableIndex ?? `new-${index}`}`}
                className="flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3"
              >
                <div className="mb-3 flex items-center justify-between gap-2">
                  <NewActivityLabel players={players} playerId={turnActivity?.playerId} />
                  <Badge variant="outline">{composition.cards.length} cards</Badge>
                </div>
                <div className="flex items-start gap-2">
                  {draftCardInstances(composition.cards).map(({ card, key }) => (
                    <GameCard
                      key={`${composition.tableIndex ?? "new"}-${key}`}
                      card={card}
                      size="default"
                    />
                  ))}
                </div>
              </div>
            ))}

            {newCompositions.map((composition) => (
              <GameBoardDraftDropZone
                key={composition.id}
                id={draftCompositionDropId(composition.id)}
                className="flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3"
              >
                <div className="mb-3 flex items-center justify-between gap-2">
                  <NewActivityLabel players={players} />
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
                        size="default"
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
          </div>
        </div>
      </CardContent>
      <CardFooter
        className={cn(
          "min-h-16 justify-center transition-opacity duration-150",
          hasDraftedCompositions ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      >
        <div className="flex items-center gap-2">
          <Button type="button" size="sm" variant="ghost" onClick={onResetTablePlay}>
            Reset
          </Button>
          <Button
            type="button"
            size="default"
            onClick={onSubmitTablePlay}
            disabled={!canSubmitTablePlay}
          >
            Submit table play
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}

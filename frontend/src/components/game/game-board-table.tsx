import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  buildHandEntries,
  type DraftedCompositionView,
  type HandEntry,
  type PlannedJokerReclaim,
  type TableCompositionView,
  draftCompositionDropId,
  planTableJokerReclaims,
} from "#/components/game/game-board-view-state";
import {
  type CompositionActivitySnapshot,
  type DraftCompositionSnapshot,
  type PlayerSnapshot,
  type TurnActivitySnapshot,
} from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import {
  draftCompositionPointTotal,
  draftCompositionPreviewPointTotal,
} from "#/components/game/game-card-utils";
import { NewActivityLabel } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Card, CardContent } from "#/components/ui/card";
import { Caption } from "#/components/typography";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

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

function alignedEntryIndexMap(
  entries: HandEntry[],
  placements: DraftCompositionSnapshot["cardPlacements"],
  field: "insertIndex" | "reclaimJokerIndex",
) {
  const normalized: Record<string, number> = {};

  for (const [index, entry] of entries.entries()) {
    const value = placements?.[index]?.[field];
    if (value !== undefined) {
      normalized[entry.key] = value;
    }
  }

  return normalized;
}

export function draftPreviewForComposition(
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
      cardInsertIndices: undefined,
    };
  }

  const insertIndex = draft.insertIndex ?? tableComposition.snapshot.cards.length;
  const normalizedReclaimTargets = alignedEntryIndexMap(
    stagedEntries,
    draft.cardPlacements,
    "reclaimJokerIndex",
  );
  const normalizedCardInsertIndices = alignedEntryIndexMap(
    stagedEntries,
    draft.cardPlacements,
    "insertIndex",
  );

  const plannedReclaims = planTableJokerReclaims(
    tableComposition.snapshot,
    stagedEntries,
    insertIndex,
    normalizedReclaimTargets,
  );
  const reclaims = plannedReclaims.reclaims;
  const reclaimedEntryKeys = plannedReclaims.reclaimedEntryKeys;
  const additionEntries = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));
  const entriesByInsertIndex = new Map<number, HandEntry[]>();

  for (const entry of additionEntries) {
    const entryInsertIndex = normalizedCardInsertIndices[entry.key] ?? insertIndex;
    const entries = entriesByInsertIndex.get(entryInsertIndex) ?? [];
    entries.push(entry);
    entriesByInsertIndex.set(entryInsertIndex, entries);
  }

  const orderedAdditionEntries = [
    ...[...(entriesByInsertIndex.get(0) ?? [])].reverse(),
    ...[...entriesByInsertIndex]
      .filter(([entryInsertIndex]) => entryInsertIndex !== 0)
      .flatMap(([, entries]) => entries),
  ];

  return {
    stagedEntries: orderedAdditionEntries,
    reclaims,
    insertIndex,
    cardInsertIndices: normalizedCardInsertIndices,
  };
}

export function GameBoardTable({
  tableCompositions,
  newCompositions,
  players,
  turnActivity,
  canCompose,
  showDraftTotal,
  invalidCompositionIds = new Set<string>(),
  invalidEntryKeys = new Set<string>(),
}: {
  tableCompositions: TableCompositionView[];
  newCompositions: DraftedCompositionView[];
  players: PlayerSnapshot[];
  turnActivity?: TurnActivitySnapshot;
  canCompose: boolean;
  showDraftTotal: boolean;
  invalidCompositionIds?: Set<string>;
  invalidEntryKeys?: Set<string>;
}) {
  const { active } = useDndContext();
  const { setNodeRef, isOver: isOverNewCompositionBoard } = useDroppable({
    id: NEW_COMPOSITION_DROP_ID,
    disabled: !canCompose,
  });
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

  const additionPointTotals = tableCompositions.flatMap((composition) => {
    const spectatorDraft = spectatorDraftsByTableIndex.get(composition.tableIndex);
    const stagedEntries =
      composition.stagedEntries.length > 0
        ? composition.stagedEntries
        : (spectatorDraft?.stagedEntries ?? []);
    const reclaims =
      composition.reclaims.length > 0 ? composition.reclaims : (spectatorDraft?.reclaims ?? []);
    const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
    const additions = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));

    if (additions.length === 0) {
      return [];
    }

    const previewPoints = draftCompositionPreviewPointTotal(
      composition.snapshot,
      additions.map((entry) => entry.card),
      reclaims.map((reclaim) => ({
        jokerIndex: reclaim.jokerIndex,
        replacementCard: reclaim.replacementEntry.card,
      })),
    );

    return [
      previewPoints === null ? null : Math.max(0, previewPoints - composition.snapshot.points),
    ];
  });
  const visibleDraftPointTotals = [
    ...stagedNewDrafts.map((composition) => draftCompositionPointTotal(composition.cards)),
    ...newCompositions.map((composition) =>
      draftCompositionPointTotal(composition.entries.map((entry) => entry.card)),
    ),
    ...additionPointTotals,
  ];
  const visibleDraftPointsTotal = visibleDraftPointTotals.every(
    (points): points is number => points !== null,
  )
    ? visibleDraftPointTotals.reduce((total, points) => total + points, 0)
    : null;

  return (
    <Card className="min-h-0 overflow-hiddden overflow-y-auto xl:flex-1">
      <CardContent className="min-h-0 flex-1">
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
                          invalidEntryKeys={invalidEntryKeys}
                        />
                      );
                    })()}
                  </div>
                ))}
              </div>
            ) : (
              <Caption className="mx-auto grid min-h-32 w-fit max-w-full place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center">
                {m.no_compositions()}
              </Caption>
            )}
          </div>

          <div className="flex flex-wrap items-center justify-center gap-3">
            {showDraftTotal && visibleDraftPointTotals.length > 0 ? (
              <div className="flex min-h-5 basis-full items-center justify-center">
                <Badge variant="outline">
                  {m.draft_total()}{" "}
                  {visibleDraftPointsTotal === null ? (
                    <span title={m.complete_drafts_points()}>?</span>
                  ) : (
                    <AnimatedNumber value={visibleDraftPointsTotal} />
                  )}{" "}
                  {m.points_unit()}
                </Badge>
              </div>
            ) : null}

            {stagedNewDrafts.map((composition: DraftCompositionSnapshot, index: number) => (
              <div
                key={`turn-draft-${composition.tableIndex ?? `new-${index}`}`}
                className="flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3"
              >
                <div className="mb-2.5 flex min-h-5 items-center justify-between gap-2">
                  <NewActivityLabel players={players} playerId={turnActivity?.playerId} />
                  <Badge variant="outline">
                    {draftCompositionPointTotal(composition.cards) === null ? (
                      <span title={m.complete_composition_points()}>?</span>
                    ) : (
                      <AnimatedNumber value={draftCompositionPointTotal(composition.cards) ?? 0} />
                    )}{" "}
                    {m.points_unit()}
                  </Badge>
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
                className={cn(
                  "flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3",
                  invalidCompositionIds.has(composition.id)
                    ? "border-destructive bg-destructive/5 ring-1 ring-destructive/30"
                    : null,
                )}
                invalid={invalidCompositionIds.has(composition.id)}
              >
                <div className="mb-2.5 flex min-h-5 items-center justify-between gap-2">
                  <NewActivityLabel players={players} />
                  <Badge variant="outline">
                    {draftCompositionPointTotal(composition.entries.map((entry) => entry.card)) ===
                    null ? (
                      <span title={m.complete_composition_points()}>?</span>
                    ) : (
                      <AnimatedNumber
                        value={
                          draftCompositionPointTotal(
                            composition.entries.map((entry) => entry.card),
                          ) ?? 0
                        }
                      />
                    )}{" "}
                    {m.points_unit()}
                  </Badge>
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
    </Card>
  );
}

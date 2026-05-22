import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  buildHandEntries,
  inferPlannedJokerReclaims,
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
    };
  }

  const insertIndex = draft.insertIndex ?? tableComposition.snapshot.cards.length;
  const { reclaims } = inferPlannedJokerReclaims(
    tableComposition.snapshot,
    stagedEntries,
    insertIndex,
  );
  return { stagedEntries, reclaims, insertIndex };
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
      playerId?: string;
    }
  >();
  const stagedNewDrafts = stagedDrafts.filter(
    (composition) => composition.tableIndex === undefined,
  );

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
    <Card className="min-h-0 overflow-y-auto xl:flex-1">
      <CardHeader>
        <CardTitle>Table</CardTitle>
        <CardAction>
          <Badge variant="outline">{tablePoints} pts</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="min-h-0 flex-1">
        <div className="flex min-h-full flex-col justify-center gap-4">
          <div className="min-h-0">
            {hasVisibleCompositions ? (
              <div className="flex min-h-0 flex-wrap items-center justify-center gap-3">
                {tableCompositions.map((composition) => (
                  <div key={composition.key} className="w-fit shrink-0">
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

                      return (
                        <CompositionRow
                          composition={composition.snapshot}
                          index={composition.tableIndex}
                          stagedEntries={stagedEntries}
                          reclaims={reclaims}
                          insertIndex={insertIndex}
                          players={players}
                          stagedEntryPlayerId={spectatorDraft?.playerId}
                          stagedEntriesInteractive={composition.stagedEntries.length > 0}
                          activity={activityByIndex.get(composition.tableIndex)}
                        />
                      );
                    })()}
                  </div>
                ))}
              </div>
            ) : (
              <div className="grid min-h-32 place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center text-sm text-muted-foreground">
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

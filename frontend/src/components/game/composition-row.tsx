import { useDndContext } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { Tick01FreeIcons } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type HandEntry,
  type PlannedJokerReclaim,
  tableCompositionDropId,
  tableCompositionEdgeDropId,
} from "#/components/game/game-board-view-state";
import {
  type CompositionSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import {
  ActivityLabel,
  NewActivityLabel,
  ReclaimActivityLabel,
} from "#/components/game/game-view-utils";
import { cn } from "#/lib/utils";
import { Badge } from "../ui/badge";

const EMPTY_STAGED_ENTRIES: HandEntry[] = [];
const EMPTY_RECLAIMS: PlannedJokerReclaim[] = [];

function compositionCardKey(card: CompositionSnapshot["cards"][number]) {
  if (card.isJoker) {
    return "joker";
  }

  return `${card.rank ?? "?"}-${card.suit ?? "?"}`;
}

function buildCompositionCardViews(cards: CompositionSnapshot["cards"]) {
  const counts = new Map<string, number>();
  const cardViews: Array<{ card: (typeof cards)[number]; index: number; key: string }> = [];

  for (const [index, card] of cards.entries()) {
    const baseKey = compositionCardKey(card);
    const occurrence = (counts.get(baseKey) ?? 0) + 1;
    counts.set(baseKey, occurrence);
    cardViews.push({ card, index, key: `${baseKey}-${occurrence}` });
  }

  return cardViews;
}

function CompositionEdgeDraftZone({
  entries,
  interactive,
  players,
  playerId,
}: {
  entries: HandEntry[];
  interactive: boolean;
  players: PlayerSnapshot[];
  playerId?: string;
}) {
  return (
    <SortableContext
      items={entries.map((entry) => entry.key)}
      strategy={horizontalListSortingStrategy}
    >
      <div className="flex shrink-0 items-center gap-2">
        {entries.map((entry) => (
          <GameCard
            key={entry.key}
            card={entry.card}
            size="default"
            draggable={
              interactive
                ? {
                    id: entry.key,
                    cardIndex: entry.sourceIndex,
                    isVirtual: entry.isVirtual,
                  }
                : undefined
            }
            decoration={{
              highlight: "addition",
              label: <ActivityLabel players={players} playerId={playerId} label="Add" />,
            }}
          />
        ))}
      </div>
    </SortableContext>
  );
}

function CompositionEdgeDropTarget({
  compositionIndex,
  edge,
  visible,
}: {
  compositionIndex: number;
  edge: "start" | "end";
  visible: boolean;
}) {
  return (
    <GameBoardDraftDropZone
      id={tableCompositionEdgeDropId(compositionIndex, edge)}
      activeClassName={null}
      className={cn(
        "pointer-events-none absolute top-1/2 z-10 flex h-24 w-10 -translate-y-1/2 items-center justify-center opacity-0 transition-opacity duration-150 data-[over=true]:opacity-100 [&[data-over=true]_.edge-drop-line]:bg-primary/70 [&[data-over=true]_.edge-drop-pill]:border-primary/70 [&[data-over=true]_.edge-drop-pill]:bg-primary/10 [&[data-over=true]_.edge-drop-pill]:text-primary",
        edge === "start" ? "-left-5" : "-right-5",
        visible ? "pointer-events-auto opacity-100" : null,
      )}
    >
      <div className="pointer-events-none relative flex h-full w-full items-center justify-center">
        <div className="edge-drop-line h-16 w-px rounded-full bg-border/60 shadow-[0_0_0_1px_hsl(var(--background))] transition-colors duration-150" />
        <div className="edge-drop-pill absolute flex size-5 items-center justify-center rounded-full border border-border/60 bg-background/85 text-[0.7rem] leading-none text-muted-foreground shadow-sm backdrop-blur-sm transition-[border-color,color,background-color] duration-150">
          +
        </div>
      </div>
    </GameBoardDraftDropZone>
  );
}

export function CompositionRow({
  composition,
  index,
  stagedEntries = EMPTY_STAGED_ENTRIES,
  reclaims = EMPTY_RECLAIMS,
  insertIndex,
  cardInsertIndices,
  players,
  stagedEntryPlayerId,
  stagedEntriesInteractive = true,
  activity,
}: {
  composition: CompositionSnapshot;
  index: number;
  stagedEntries?: HandEntry[];
  reclaims?: PlannedJokerReclaim[];
  insertIndex?: number;
  cardInsertIndices?: Record<string, number>;
  players: PlayerSnapshot[];
  stagedEntryPlayerId?: string;
  stagedEntriesInteractive?: boolean;
  activity?: {
    kind?: string;
    playerId?: string;
    cardActivities?: Record<number, { kind: string; playerId: string }>;
  };
}) {
  const { active, over } = useDndContext();
  const compositionCards = buildCompositionCardViews(composition.cards);
  const reclaimByJokerIndex = new Map(reclaims.map((reclaim) => [reclaim.jokerIndex, reclaim]));
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  const additionEntries = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));
  const cardActivities = activity?.cardActivities ?? {};
  const isNewComposition = activity?.kind === "new_composition";
  const isHighlightedComposition = isNewComposition || composition.complete;

  function entryInsertIndex(entry: HandEntry): number {
    return cardInsertIndices?.[entry.key] ?? insertIndex ?? composition.cards.length;
  }

  const additionsAtStart = additionEntries.filter((entry) => entryInsertIndex(entry) === 0);
  const additionsAtEnd = additionEntries.filter((entry) => entryInsertIndex(entry) !== 0);
  const compositionDropId = tableCompositionDropId(index);
  const startEdgeDropId = tableCompositionEdgeDropId(index, "start");
  const endEdgeDropId = tableCompositionEdgeDropId(index, "end");
  const overId = over ? String(over.id) : null;
  const isDraggingCompositionCard =
    active !== null &&
    active.data.current?.drawSource === undefined &&
    typeof active.id === "string";
  const isDraggingOverComposition =
    isDraggingCompositionCard &&
    (overId === compositionDropId || overId === startEdgeDropId || overId === endEdgeDropId);
  const showStartEdgeDropTarget = isDraggingOverComposition;
  const showEndEdgeDropTarget = isDraggingOverComposition;

  return (
    <GameBoardDraftDropZone
      id={compositionDropId}
      className={cn(
        "relative flex min-w-0 flex-col rounded-3xl border border-border/70 bg-muted/20 p-4",
        isHighlightedComposition ? "mt-5 border-primary/70 bg-primary/5" : null,
      )}
    >
      {composition.complete ? (
        <div className="absolute -top-1 -right-1 z-10 flex size-5 translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border border-primary/70 bg-primary/5 text-primary/70 shadow-sm">
          <HugeiconsIcon icon={Tick01FreeIcons} className="size-3" strokeWidth={2} />
        </div>
      ) : null}

      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">#{index + 1}</Badge>
          {isNewComposition ? (
            <NewActivityLabel players={players} playerId={activity?.playerId} />
          ) : null}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{composition.points} pts</span>
        </div>
      </div>

      <div className="relative flex flex-wrap items-center justify-center gap-3">
        <CompositionEdgeDropTarget
          compositionIndex={index}
          edge="start"
          visible={showStartEdgeDropTarget}
        />

        {additionsAtStart.length > 0 ? (
          <CompositionEdgeDraftZone
            entries={additionsAtStart}
            interactive={stagedEntriesInteractive}
            players={players}
            playerId={stagedEntryPlayerId}
          />
        ) : null}

        {compositionCards.map(({ card, index: cardIndex, key }) => {
          const reclaim = reclaimByJokerIndex.get(cardIndex);
          const cardActivity = cardActivities[cardIndex];
          const submittedHighlight =
            !isNewComposition && cardActivity
              ? cardActivity.kind === "joker_reclaim"
                ? "joker_reclaim"
                : "addition"
              : undefined;
          const previewPlayerId = cardActivity?.playerId ?? stagedEntryPlayerId;
          const previewCard = reclaim ? reclaim.replacementEntry.card : card;

          return (
            <div key={key} className="flex flex-col items-center gap-1">
              <GameCard
                card={previewCard}
                size="default"
                draggable={
                  reclaim && stagedEntriesInteractive
                    ? {
                        id: reclaim.replacementEntry.key,
                        cardIndex: reclaim.replacementEntry.sourceIndex,
                        isVirtual: reclaim.replacementEntry.isVirtual,
                      }
                    : undefined
                }
                decoration={
                  submittedHighlight || reclaim
                    ? {
                        highlight: submittedHighlight ?? "joker_reclaim",
                        label:
                          (submittedHighlight ?? "joker_reclaim") === "joker_reclaim" ? (
                            <ReclaimActivityLabel players={players} playerId={previewPlayerId} />
                          ) : (
                            <ActivityLabel
                              players={players}
                              playerId={previewPlayerId}
                              label="Add"
                            />
                          ),
                      }
                    : undefined
                }
              />
            </div>
          );
        })}

        {additionsAtEnd.length > 0 ? (
          <CompositionEdgeDraftZone
            entries={additionsAtEnd}
            interactive={stagedEntriesInteractive}
            players={players}
            playerId={stagedEntryPlayerId}
          />
        ) : null}

        <CompositionEdgeDropTarget
          compositionIndex={index}
          edge="end"
          visible={showEndEdgeDropTarget}
        />
      </div>
    </GameBoardDraftDropZone>
  );
}

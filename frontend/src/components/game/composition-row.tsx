import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { Tick01FreeIcons } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  canReclaimJokerWithCard,
  type HandEntry,
  type PlannedJokerReclaim,
  tableCompositionDropId,
  tableCompositionEdgeDropId,
  tableCompositionJokerDropId,
} from "#/components/game/game-board-view-state";
import {
  type CompositionSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { draftCompositionPreviewPointTotal } from "#/components/game/game-card-utils";
import {
  ActivityLabel,
  NewActivityLabel,
  ReclaimActivityLabel,
} from "#/components/game/game-view-utils";
import { cn } from "#/lib/utils";
import { Badge } from "../ui/badge";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Text } from "#/components/typography";

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
  invalidEntryKeys,
}: {
  entries: HandEntry[];
  interactive: boolean;
  players: PlayerSnapshot[];
  playerId?: string;
  invalidEntryKeys: Set<string>;
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
              label: (
                <ActivityLabel
                  players={players}
                  playerId={playerId}
                  label="Add"
                  offsetClassName="translate-y-[2px]"
                />
              ),
            }}
            invalid={invalidEntryKeys.has(entry.key)}
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
  active,
}: {
  compositionIndex: number;
  edge: "start" | "end";
  visible: boolean;
  active: boolean;
}) {
  const edgeDropId = tableCompositionEdgeDropId(compositionIndex, edge);

  return (
    <GameBoardDraftDropZone
      id={edgeDropId}
      activeClassName={null}
      className={cn(
        "pointer-events-none absolute top-1/2 z-20 flex h-28 w-12 -translate-y-1/2 items-center justify-center overflow-visible",
        edge === "start" ? "left-0 -translate-x-1/2" : "right-0 translate-x-1/2",
        active ? "pointer-events-auto" : null,
      )}
    >
      <div className="pointer-events-none relative flex h-full w-full items-center justify-center">
        <div
          className={cn(
            "edge-drop-line absolute top-1/2 h-16 w-0.5 -translate-x-1/2 -translate-y-1/2 rounded-full shadow-[0_0_0_1px_hsl(var(--background))] transition-colors duration-150",
            edge === "start" ? "left-1/2" : "left-1/2",
            active ? "bg-primary/80" : "bg-transparent",
          )}
        />
        <Text
          as="div"
          variant="symbol"
          className={cn(
            "edge-drop-pill absolute flex size-5 items-center justify-center rounded-full shadow-sm backdrop-blur-sm transition-[border-color,color,background-color,opacity] duration-150",
            "left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
            visible ? "opacity-100" : "opacity-0",
            active
              ? "border border-primary/80 bg-primary text-primary-foreground opacity-100"
              : "border border-border/60 bg-background/90 text-muted-foreground",
          )}
        >
          +
        </Text>
      </div>
    </GameBoardDraftDropZone>
  );
}

function JokerReclaimDropTarget({
  compositionIndex,
  jokerIndex,
  active,
  visible,
}: {
  compositionIndex: number;
  jokerIndex: number;
  active: boolean;
  visible: boolean;
}) {
  const jokerDropId = tableCompositionJokerDropId(compositionIndex, jokerIndex);

  return (
    <GameBoardDraftDropZone
      id={jokerDropId}
      activeClassName={null}
      className={cn(
        "pointer-events-none absolute inset-0 z-10 rounded-2xl transition-colors",
        active ? "pointer-events-auto" : null,
      )}
    >
      <div
        className={cn(
          "flex h-full w-full items-center justify-center rounded-2xl border-2 border-dashed backdrop-blur-[1px] transition-[border-color,background-color,opacity,transform] duration-150",
          visible ? "opacity-100" : "opacity-0",
          active
            ? "scale-[1.02] border-primary/80 bg-primary/14 shadow-[0_0_0_1px_hsl(var(--primary)/0.2)]"
            : "border-primary/35 bg-primary/6",
        )}
      />
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
  dropTargetsEnabled = false,
  activity,
  invalidEntryKeys = new Set<string>(),
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
  dropTargetsEnabled?: boolean;
  activity?: {
    kind?: string;
    playerId?: string;
    cardActivities?: Record<number, { kind: string; playerId: string }>;
  };
  invalidEntryKeys?: Set<string>;
}) {
  const { active, over } = useDndContext();
  const compositionDropId = tableCompositionDropId(index);
  const { setNodeRef: setCompositionHoverRef, isOver: isOverComposition } = useDroppable({
    id: `${compositionDropId}-hover`,
  });
  const compositionCards = buildCompositionCardViews(composition.cards);
  const reclaimByJokerIndex = new Map(reclaims.map((reclaim) => [reclaim.jokerIndex, reclaim]));
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  const additionEntries = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));
  const hasPointPreview = additionEntries.length > 0 || reclaims.length > 0;
  const previewPoints = hasPointPreview
    ? draftCompositionPreviewPointTotal(
        composition,
        additionEntries.map((entry) => entry.card),
        reclaims.map((reclaim) => ({
          jokerIndex: reclaim.jokerIndex,
          replacementCard: reclaim.replacementEntry.card,
        })),
      )
    : composition.points;
  const cardActivities = activity?.cardActivities ?? {};
  const isNewComposition = activity?.kind === "new_composition";
  const isHighlightedComposition = isNewComposition;

  function entryInsertIndex(entry: HandEntry): number {
    return cardInsertIndices?.[entry.key] ?? insertIndex ?? composition.cards.length;
  }

  const additionsAtStart = additionEntries.filter((entry) => entryInsertIndex(entry) === 0);
  const additionsAtEnd = additionEntries.filter((entry) => entryInsertIndex(entry) !== 0);
  const startEdgeDropId = tableCompositionEdgeDropId(index, "start");
  const endEdgeDropId = tableCompositionEdgeDropId(index, "end");
  const overId = over ? String(over.id) : null;
  const isDraggingCompositionCard =
    active !== null &&
    active.data.current?.drawSource === undefined &&
    typeof active.id === "string";
  const jokerDropIdPrefix = `${tableCompositionJokerDropId(index, 0).slice(0, -1)}`;
  const highlightStartEdgeDropTarget = isDraggingCompositionCard && overId === startEdgeDropId;
  const highlightEndEdgeDropTarget = isDraggingCompositionCard && overId === endEdgeDropId;
  const isHoveringAnyCompositionTarget =
    isDraggingCompositionCard &&
    (isOverComposition ||
      overId === compositionDropId ||
      overId === startEdgeDropId ||
      overId === endEdgeDropId ||
      overId?.startsWith(jokerDropIdPrefix) === true);
  return (
    <GameBoardDraftDropZone
      id={compositionDropId}
      activeClassName={null}
      className={cn(
        "relative flex min-w-0 flex-col overflow-visible rounded-3xl border border-border/70 bg-muted/20 p-3",
        isHighlightedComposition ? "border-primary/70 bg-primary/5" : null,
      )}
    >
      <div
        ref={setCompositionHoverRef}
        className="pointer-events-none absolute inset-0 z-0 rounded-3xl"
      />
      {composition.complete ? (
        <div className="absolute -top-1 -right-1 z-10 flex size-5 translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border border-primary/70 bg-primary/5 text-primary/70 shadow-sm">
          <HugeiconsIcon icon={Tick01FreeIcons} className="size-3" strokeWidth={2} />
        </div>
      ) : null}

      <div className="mb-2.5 flex min-h-5 flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">#{index + 1}</Badge>
          {isNewComposition ? (
            <NewActivityLabel players={players} playerId={activity?.playerId} />
          ) : null}
        </div>
        <Text as="div" variant="caption" className="flex items-center gap-2">
          {previewPoints === null ? (
            <span title="Add a natural card to resolve this joker's composition value">?</span>
          ) : (
            <AnimatedNumber value={previewPoints} />
          )}{" "}
          pts
        </Text>
      </div>

      <div className="relative mx-auto flex w-fit max-w-full flex-wrap items-center justify-center gap-3 overflow-visible">
        <CompositionEdgeDropTarget
          compositionIndex={index}
          edge="start"
          visible={isHoveringAnyCompositionTarget}
          active={highlightStartEdgeDropTarget}
        />

        {additionsAtStart.length > 0 ? (
          <CompositionEdgeDraftZone
            entries={additionsAtStart}
            interactive={stagedEntriesInteractive}
            players={players}
            playerId={stagedEntryPlayerId}
            invalidEntryKeys={invalidEntryKeys}
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
              <div className="relative">
                {card.isJoker && dropTargetsEnabled && isDraggingCompositionCard
                  ? (() => {
                      const draggedCard = active?.data.current?.card as
                        | HandEntry["card"]
                        | undefined;
                      const canTargetJoker = draggedCard
                        ? canReclaimJokerWithCard(composition, cardIndex, draggedCard)
                        : false;
                      const jokerDropId = tableCompositionJokerDropId(index, cardIndex);
                      const showJokerDropTarget = canTargetJoker && isHoveringAnyCompositionTarget;

                      return (
                        <JokerReclaimDropTarget
                          compositionIndex={index}
                          jokerIndex={cardIndex}
                          visible={showJokerDropTarget}
                          active={canTargetJoker && overId === jokerDropId}
                        />
                      );
                    })()
                  : null}

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
                              <ReclaimActivityLabel
                                players={players}
                                playerId={previewPlayerId}
                                offsetClassName="translate-y-[2px]"
                              />
                            ) : (
                              <ActivityLabel
                                players={players}
                                playerId={previewPlayerId}
                                label="Add"
                                offsetClassName="translate-y-[2px]"
                              />
                            ),
                        }
                      : undefined
                  }
                  invalid={Boolean(reclaim && invalidEntryKeys.has(reclaim.replacementEntry.key))}
                />
              </div>
            </div>
          );
        })}

        {additionsAtEnd.length > 0 ? (
          <CompositionEdgeDraftZone
            entries={additionsAtEnd}
            interactive={stagedEntriesInteractive}
            players={players}
            playerId={stagedEntryPlayerId}
            invalidEntryKeys={invalidEntryKeys}
          />
        ) : null}

        <CompositionEdgeDropTarget
          compositionIndex={index}
          edge="end"
          visible={isHoveringAnyCompositionTarget}
          active={highlightEndEdgeDropTarget}
        />
      </div>
    </GameBoardDraftDropZone>
  );
}

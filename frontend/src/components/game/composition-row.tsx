import { keyedCards, type HandEntry } from "#/components/game/card";
import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { AnimatePresence, motion } from "motion/react";
import {
  tableCompositionDropId,
  tableCompositionEdgeDropId,
  tableCompositionJokerDropId,
  tableCompositionJokerTargetFromDropId,
} from "#/components/game/board-drop-targets";
import { type CompositionSnapshot, type PlayerSnapshot } from "#/lib/game-protocol";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { isCompleteDraftComposition } from "#/components/game/game-card-utils";
import {
  canReclaimJokerWithCard,
  cardsWithTableDraft,
  type TableDraftPlan,
} from "#/components/game/table-draft";
import {
  AddActivityLabel,
  NewActivityLabel,
  ReclaimActivityLabel,
} from "#/components/game/game-view-utils";
import {
  spectatorCardEnter,
  spectatorCardExit,
  spectatorCardExitTransition,
  spectatorCardTransition,
  spectatorCardVisible,
} from "#/components/game/spectator-card-motion";
import { cn } from "#/lib/utils";

function CompositionEdgeDraftZone({
  entries,
  interactive,
  animate,
  players,
  playerId,
  invalidEntryKeys,
  complete,
  overlapFirst,
}: {
  entries: HandEntry[];
  interactive: boolean;
  animate: boolean;
  players: PlayerSnapshot[];
  playerId?: string;
  invalidEntryKeys: Set<string>;
  complete: boolean;
  overlapFirst: boolean;
}) {
  return (
    <SortableContext
      items={entries.map((entry) => entry.key)}
      strategy={horizontalListSortingStrategy}
    >
      <div
        data-slot="composition-edge-draft-zone"
        className={cn(
          entries.length === 0
            ? "contents"
            : complete
              ? "contents"
              : "flex shrink-0 items-center gap-2",
        )}
      >
        {interactive
          ? entries.map((entry, entryIndex) => (
              <div
                key={entry.key}
                data-composition-card-wrap
                data-card-rank={entry.card.rank}
                data-card-suit={entry.card.suit}
                data-card-joker={entry.card.isJoker || undefined}
                data-composition-card-overlap={
                  complete && (overlapFirst || entryIndex > 0) ? "true" : undefined
                }
              >
                <GameCard
                  card={entry.card}
                  size="default"
                  draggable={{
                    id: entry.key,
                    cardIndex: entry.sourceIndex,
                    isVirtual: entry.isVirtual,
                  }}
                  decoration={{
                    highlight: "addition",
                    label: (
                      <AddActivityLabel
                        players={players}
                        playerId={playerId}
                        offsetClassName="translate-y-[2px]"
                      />
                    ),
                  }}
                  invalid={invalidEntryKeys.has(entry.key)}
                />
              </div>
            ))
          : null}
        <AnimatePresence initial={false} mode="popLayout">
          {animate && !interactive
            ? entries.map((entry, entryIndex) => (
                <motion.div
                  key={entry.key}
                  layout="position"
                  initial={spectatorCardEnter}
                  animate={spectatorCardVisible}
                  exit={{
                    ...spectatorCardExit,
                    transition: spectatorCardExitTransition,
                  }}
                  transition={spectatorCardTransition}
                  data-spectator-card-motion="addition"
                  data-composition-card-wrap
                  data-card-rank={entry.card.rank}
                  data-card-suit={entry.card.suit}
                  data-card-joker={entry.card.isJoker || undefined}
                  data-composition-card-overlap={
                    complete && (overlapFirst || entryIndex > 0) ? "true" : undefined
                  }
                >
                  <GameCard
                    card={entry.card}
                    size="default"
                    decoration={{
                      highlight: "addition",
                      label: (
                        <AddActivityLabel
                          players={players}
                          playerId={playerId}
                          offsetClassName="translate-y-[2px]"
                        />
                      ),
                    }}
                    invalid={invalidEntryKeys.has(entry.key)}
                  />
                </motion.div>
              ))
            : null}
        </AnimatePresence>
        {!animate && !interactive
          ? entries.map((entry, entryIndex) => (
              <motion.div
                key={entry.key}
                layout="position"
                transition={spectatorCardTransition}
                data-composition-card-wrap
                data-card-rank={entry.card.rank}
                data-card-suit={entry.card.suit}
                data-card-joker={entry.card.isJoker || undefined}
                data-composition-card-overlap={
                  complete && (overlapFirst || entryIndex > 0) ? "true" : undefined
                }
              >
                <GameCard
                  card={entry.card}
                  size="default"
                  decoration={{
                    highlight: "addition",
                    label: (
                      <AddActivityLabel
                        players={players}
                        playerId={playerId}
                        offsetClassName="translate-y-[2px]"
                      />
                    ),
                  }}
                  invalid={invalidEntryKeys.has(entry.key)}
                />
              </motion.div>
            ))
          : null}
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
        <span
          className={cn(
            "edge-drop-pill absolute flex size-5 items-center justify-center rounded-full shadow-sm backdrop-blur-sm transition-[border-color,color,background-color,opacity] duration-150",
            "top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2",
            visible ? "opacity-100" : "opacity-0",
            active
              ? "border border-primary/80 bg-primary text-primary-foreground opacity-100"
              : "border border-border/60 bg-background/90 text-muted-foreground",
          )}
        >
          +
        </span>
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
  draft,
  players,
  stagedEntryPlayerId,
  stagedEntriesInteractive = true,
  animateStagedEntries = !stagedEntriesInteractive,
  dropTargetsEnabled = false,
  activity,
  invalidEntryKeys = new Set<string>(),
}: {
  composition: CompositionSnapshot;
  index: number;
  draft?: TableDraftPlan;
  players: PlayerSnapshot[];
  stagedEntryPlayerId?: string;
  stagedEntriesInteractive?: boolean;
  animateStagedEntries?: boolean;
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
  const compositionCards = keyedCards(composition.cards);
  const reclaimByJokerIndex = new Map(
    draft?.reclaims.map((reclaim) => [reclaim.jokerIndex, reclaim]),
  );
  const cardActivities = activity?.cardActivities ?? {};
  const isNewComposition = activity?.kind === "new_composition";
  const isHighlightedComposition = isNewComposition;

  const additionsAtStart =
    draft?.additions.find((addition) => addition.insertIndex === 0)?.entries ?? [];
  const additionsAtEnd =
    draft?.additions.flatMap((addition) => (addition.insertIndex === 0 ? [] : addition.entries)) ??
    [];
  const complete =
    (!draft && composition.complete) ||
    isCompleteDraftComposition(
      draft ? cardsWithTableDraft(composition, draft) : composition.cards,
      composition.type,
    );
  const startEdgeDropId = tableCompositionEdgeDropId(index, "start");
  const endEdgeDropId = tableCompositionEdgeDropId(index, "end");
  const overId = over ? String(over.id) : null;
  const isDraggingCompositionCard =
    active !== null &&
    active.data.current?.drawSource === undefined &&
    typeof active.id === "string";
  const jokerDropTarget = overId ? tableCompositionJokerTargetFromDropId(overId) : null;
  const highlightStartEdgeDropTarget = isDraggingCompositionCard && overId === startEdgeDropId;
  const highlightEndEdgeDropTarget = isDraggingCompositionCard && overId === endEdgeDropId;
  const isHoveringAnyCompositionTarget =
    isDraggingCompositionCard &&
    (isOverComposition ||
      overId === compositionDropId ||
      overId === startEdgeDropId ||
      overId === endEdgeDropId ||
      jokerDropTarget?.compositionIndex === index);
  return (
    <GameBoardDraftDropZone
      id={compositionDropId}
      activeClassName={null}
      completedComposition={complete}
      className={cn(
        "relative flex min-w-0 flex-col overflow-visible rounded-3xl border border-border/70 bg-muted/20 p-2 xl:p-3",
        isHighlightedComposition ? "border-primary/70 bg-primary/5" : null,
      )}
    >
      <div
        ref={setCompositionHoverRef}
        className="pointer-events-none absolute inset-0 z-0 rounded-3xl"
      />
      {isNewComposition ? (
        <div className="mb-1.5 flex min-h-5 items-center xl:mb-2.5">
          <NewActivityLabel players={players} playerId={activity?.playerId} />
        </div>
      ) : null}

      <div className="relative w-fit max-w-none overflow-visible xl:mx-auto xl:max-w-full">
        <CompositionEdgeDropTarget
          compositionIndex={index}
          edge="start"
          visible={isHoveringAnyCompositionTarget}
          active={highlightStartEdgeDropTarget}
        />

        <div
          className="composition-card-track flex w-fit max-w-none flex-nowrap items-center justify-start gap-2 overflow-visible xl:max-w-full xl:flex-wrap xl:justify-center xl:gap-3"
          data-complete={complete}
        >
          {additionsAtStart.length > 0 || !stagedEntriesInteractive ? (
            <CompositionEdgeDraftZone
              entries={additionsAtStart}
              interactive={stagedEntriesInteractive}
              animate={animateStagedEntries}
              players={players}
              playerId={stagedEntryPlayerId}
              invalidEntryKeys={invalidEntryKeys}
              complete={complete}
              overlapFirst={false}
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
            const renderedCard = (
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
                            <AddActivityLabel
                              players={players}
                              playerId={previewPlayerId}
                              offsetClassName="translate-y-[2px]"
                            />
                          ),
                      }
                    : undefined
                }
                invalid={Boolean(reclaim && invalidEntryKeys.has(reclaim.replacementEntry.key))}
              />
            );

            return (
              <motion.div
                key={key}
                className="flex flex-col items-center gap-1"
                data-composition-card-wrap
                data-card-rank={previewCard.rank}
                data-card-suit={previewCard.suit}
                data-card-joker={previewCard.isJoker || undefined}
                data-composition-card-overlap={
                  complete && (additionsAtStart.length > 0 || cardIndex > 0) ? "true" : undefined
                }
              >
                <div className={cn("relative", stagedEntriesInteractive ? null : "grid")}>
                  {card.isJoker && dropTargetsEnabled && isDraggingCompositionCard
                    ? (() => {
                        const draggedCard = active?.data.current?.card as
                          | HandEntry["card"]
                          | undefined;
                        const canTargetJoker = draggedCard
                          ? canReclaimJokerWithCard(composition, cardIndex, draggedCard)
                          : false;
                        const jokerDropId = tableCompositionJokerDropId(index, cardIndex);
                        const showJokerDropTarget =
                          canTargetJoker && isHoveringAnyCompositionTarget;

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

                  {stagedEntriesInteractive || !animateStagedEntries ? renderedCard : null}
                  <AnimatePresence initial={false}>
                    {animateStagedEntries ? (
                      <motion.div
                        key={reclaim ? `reclaim-${reclaim.replacementEntry.key}` : `table-${key}`}
                        className="col-start-1 row-start-1"
                        initial={spectatorCardEnter}
                        animate={spectatorCardVisible}
                        exit={{
                          ...spectatorCardExit,
                          transition: spectatorCardExitTransition,
                        }}
                        transition={spectatorCardTransition}
                        data-spectator-card-motion={reclaim ? "joker-reclaim" : undefined}
                      >
                        {renderedCard}
                      </motion.div>
                    ) : null}
                  </AnimatePresence>
                </div>
              </motion.div>
            );
          })}

          {additionsAtEnd.length > 0 || !stagedEntriesInteractive ? (
            <CompositionEdgeDraftZone
              entries={additionsAtEnd}
              interactive={stagedEntriesInteractive}
              animate={animateStagedEntries}
              players={players}
              playerId={stagedEntryPlayerId}
              invalidEntryKeys={invalidEntryKeys}
              complete={complete}
              overlapFirst
            />
          ) : null}
        </div>

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

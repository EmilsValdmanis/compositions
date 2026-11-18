import {
  draftCompositionDropId,
  NEW_COMPOSITION_DROP_ID,
} from "#/components/game/board-drop-targets";
import { keyedCards } from "#/components/game/card";
import { Cards01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, type RefObject } from "react";
import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  type DraftedCompositionView,
  type TableCompositionView,
} from "#/components/game/game-board-view-state";
import {
  type CompositionActivitySnapshot,
  type DraftCompositionSnapshot,
  type PlayerSnapshot,
  type TurnActivitySnapshot,
} from "#/lib/game-protocol";
import { CompositionRow } from "#/components/game/composition-row";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { cardsWithTableDraft, draftPreviewForComposition } from "#/components/game/table-draft";
import {
  draftCompositionPointTotal,
  isCompleteDraftComposition,
  jokerReclaimPointValue,
} from "#/components/game/game-card-utils";
import {
  spectatorCardEnter,
  spectatorCardExit,
  spectatorCardExitTransition,
  spectatorCardTransition,
  spectatorCardVisible,
  spectatorCompositionEnter,
  spectatorCompositionExit,
} from "#/components/game/spectator-card-motion";
import { NewActivityLabel } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Card, CardContent } from "#/components/ui/card";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "#/components/ui/empty";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

type VisibleTableComposition = TableCompositionView & {
  interactive: boolean;
  draftPlayerId?: string;
};

function TableComposition({
  composition,
  players,
  activity,
  canCompose,
  viewerPlayerId,
  invalidEntryKeys,
}: {
  composition: VisibleTableComposition;
  players: PlayerSnapshot[];
  activity?: CompositionActivitySnapshot;
  canCompose: boolean;
  viewerPlayerId?: string;
  invalidEntryKeys: Set<string>;
}) {
  const changePlayerId = composition.draftPlayerId ?? activity?.playerId;
  const animateRemoteChanges = Boolean(
    !composition.interactive && changePlayerId && changePlayerId !== viewerPlayerId,
  );

  return (
    <div className="w-fit shrink-0 overflow-visible p-1">
      <CompositionRow
        composition={composition.snapshot}
        index={composition.tableIndex}
        draft={composition.draft}
        players={players}
        stagedEntryPlayerId={composition.draftPlayerId}
        stagedEntriesInteractive={composition.interactive}
        animateStagedEntries={animateRemoteChanges}
        dropTargetsEnabled={canCompose}
        activity={activity}
        invalidEntryKeys={invalidEntryKeys}
      />
    </div>
  );
}

function TableCompositionsSection({
  tableCompositions,
  activityByIndex,
  players,
  canCompose,
  viewerPlayerId,
  invalidEntryKeys,
}: {
  tableCompositions: VisibleTableComposition[];
  activityByIndex: Map<number, CompositionActivitySnapshot>;
  players: PlayerSnapshot[];
  canCompose: boolean;
  viewerPlayerId?: string;
  invalidEntryKeys: Set<string>;
}) {
  if (tableCompositions.length === 0) {
    return (
      <div className="w-full shrink-0 overflow-visible">
        <Empty className="p-6 xl:p-12">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <HugeiconsIcon icon={Cards01Icon} />
            </EmptyMedia>
            <EmptyTitle className="text-sm">{m.no_compositions()}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </div>
    );
  }

  return (
    <div className="w-full shrink-0 overflow-visible">
      <div
        data-slot="table-compositions"
        className="flex w-full flex-wrap items-center justify-center-safe gap-2 overflow-visible p-1 xl:gap-4"
      >
        {tableCompositions.map((composition) => (
          <TableComposition
            key={composition.key}
            composition={composition}
            players={players}
            activity={activityByIndex.get(composition.tableIndex)}
            canCompose={canCompose}
            viewerPlayerId={viewerPlayerId}
            invalidEntryKeys={invalidEntryKeys}
          />
        ))}
      </div>
    </div>
  );
}

function DraftTotal({ points }: { points: number | null }) {
  return (
    <div className="flex min-h-5 basis-full items-center justify-center">
      <Badge variant="outline">
        {m.draft_total()}{" "}
        {points === null ? (
          <span title={m.complete_drafts_points()}>?</span>
        ) : (
          <AnimatedNumber value={points} />
        )}{" "}
        {m.points_unit()}
      </Badge>
    </div>
  );
}

function SpectatorNewDraft({
  composition,
  players,
  playerId,
  animateCards,
}: {
  composition: DraftCompositionSnapshot;
  players: PlayerSnapshot[];
  playerId?: string;
  animateCards: boolean;
}) {
  const pointTotal = draftCompositionPointTotal(composition.cards);
  const complete = isCompleteDraftComposition(composition.cards);

  return (
    <div
      className="flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3"
      data-completed-composition={complete || undefined}
    >
      <div className="mb-2.5 flex min-h-5 items-center justify-between gap-2">
        <NewActivityLabel players={players} playerId={playerId} />
        <Badge variant="outline">
          {pointTotal === null ? (
            <span title={m.complete_composition_points()}>?</span>
          ) : (
            <AnimatedNumber value={pointTotal} />
          )}{" "}
          {m.points_unit()}
        </Badge>
      </div>
      <div className="composition-card-track flex items-start gap-2" data-complete={complete}>
        <AnimatePresence initial={false} mode="popLayout">
          {animateCards
            ? keyedCards(composition.cards).map(({ card, key }, cardIndex) => (
                <motion.div
                  key={key}
                  layout="position"
                  initial={spectatorCardEnter}
                  animate={spectatorCardVisible}
                  exit={{
                    ...spectatorCardExit,
                    transition: spectatorCardExitTransition,
                  }}
                  transition={spectatorCardTransition}
                  data-spectator-card-motion="draft"
                  data-composition-card-wrap
                  data-card-rank={card.rank}
                  data-card-suit={card.suit}
                  data-card-joker={card.isJoker || undefined}
                  data-composition-card-overlap={complete && cardIndex > 0 ? "true" : undefined}
                >
                  <GameCard card={card} size="default" />
                </motion.div>
              ))
            : null}
        </AnimatePresence>
        {!animateCards
          ? keyedCards(composition.cards).map(({ card, key }, cardIndex) => (
              <motion.div
                key={key}
                layout="position"
                transition={spectatorCardTransition}
                data-composition-card-wrap
                data-card-rank={card.rank}
                data-card-suit={card.suit}
                data-card-joker={card.isJoker || undefined}
                data-composition-card-overlap={complete && cardIndex > 0 ? "true" : undefined}
              >
                <GameCard card={card} size="default" />
              </motion.div>
            ))
          : null}
      </div>
    </div>
  );
}

function EditableNewDraft({
  composition,
  players,
  invalid,
}: {
  composition: DraftedCompositionView;
  players: PlayerSnapshot[];
  invalid: boolean;
}) {
  const pointTotal = draftCompositionPointTotal(composition.entries.map((entry) => entry.card));
  const complete = isCompleteDraftComposition(composition.entries.map((entry) => entry.card));

  return (
    <div data-onboarding-target="new-composition" className="w-fit rounded-3xl">
      <GameBoardDraftDropZone
        id={draftCompositionDropId(composition.id)}
        completedComposition={complete}
        className={cn(
          "flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3",
          invalid ? "border-destructive bg-destructive/5 ring-1 ring-destructive/30" : null,
        )}
        invalid={invalid}
      >
        <div className="mb-2.5 flex min-h-5 items-center justify-between gap-2">
          <NewActivityLabel players={players} />
          <Badge variant="outline">
            {pointTotal === null ? (
              <span title={m.complete_composition_points()}>?</span>
            ) : (
              <AnimatedNumber value={pointTotal} />
            )}{" "}
            {m.points_unit()}
          </Badge>
        </div>
        <SortableContext
          items={composition.entries.map((entry) => entry.key)}
          strategy={horizontalListSortingStrategy}
        >
          <div className="composition-card-track flex items-start gap-2" data-complete={complete}>
            {composition.entries.map((entry, entryIndex) => (
              <div
                key={entry.key}
                data-composition-card-wrap
                data-card-rank={entry.card.rank}
                data-card-suit={entry.card.suit}
                data-card-joker={entry.card.isJoker || undefined}
                data-composition-card-overlap={complete && entryIndex > 0 ? "true" : undefined}
              >
                <GameCard
                  card={entry.card}
                  size="default"
                  draggable={{
                    id: entry.key,
                    cardIndex: entry.sourceIndex,
                    isVirtual: entry.isVirtual,
                  }}
                />
              </div>
            ))}
          </div>
        </SortableContext>
      </GameBoardDraftDropZone>
    </div>
  );
}

function DraftCompositionsSection({
  sectionRef,
  showDraftTotal,
  visibleDraftPointTotals,
  visibleDraftPointsTotal,
  stagedNewDrafts,
  newCompositions,
  players,
  turnPlayerId,
  animateStagedDrafts,
  invalidCompositionIds,
  showNewCompositionDropCue,
}: {
  sectionRef: RefObject<HTMLDivElement | null>;
  showDraftTotal: boolean;
  visibleDraftPointTotals: (number | null)[];
  visibleDraftPointsTotal: number | null;
  stagedNewDrafts: DraftCompositionSnapshot[];
  newCompositions: DraftedCompositionView[];
  players: PlayerSnapshot[];
  turnPlayerId?: string;
  animateStagedDrafts: boolean;
  invalidCompositionIds: Set<string>;
  showNewCompositionDropCue: boolean;
}) {
  return (
    <div
      ref={sectionRef}
      className="flex w-full shrink-0 flex-wrap items-center justify-center gap-2 xl:gap-3"
    >
      {showNewCompositionDropCue && newCompositions.length === 0 ? (
        <div
          data-onboarding-target="new-composition"
          className="pointer-events-none grid h-24 w-40 shrink-0 place-items-center rounded-2xl border border-dashed border-foreground/30 bg-background/70 px-4 text-center text-xs font-medium text-muted-foreground shadow-sm"
        >
          <span className="flex flex-col items-center gap-1.5">
            <HugeiconsIcon icon={Cards01Icon} className="size-5 text-primary" aria-hidden="true" />
            {m.onboarding_drop_cards_here()}
          </span>
        </div>
      ) : null}
      {showDraftTotal && visibleDraftPointTotals.length > 0 ? (
        <DraftTotal points={visibleDraftPointsTotal} />
      ) : null}

      <AnimatePresence initial={false} mode="popLayout">
        {animateStagedDrafts
          ? stagedNewDrafts.map((composition) => (
              <motion.div
                key={composition.id}
                layout="position"
                initial={spectatorCompositionEnter}
                animate={spectatorCardVisible}
                exit={{
                  ...spectatorCompositionExit,
                  transition: spectatorCardExitTransition,
                }}
                transition={spectatorCardTransition}
                data-spectator-composition-motion="draft"
              >
                <SpectatorNewDraft
                  composition={composition}
                  players={players}
                  playerId={turnPlayerId}
                  animateCards
                />
              </motion.div>
            ))
          : null}
      </AnimatePresence>
      {!animateStagedDrafts
        ? stagedNewDrafts.map((composition) => (
            <SpectatorNewDraft
              key={composition.id}
              composition={composition}
              players={players}
              playerId={turnPlayerId}
              animateCards={false}
            />
          ))
        : null}

      {newCompositions.map((composition) => (
        <EditableNewDraft
          key={composition.id}
          composition={composition}
          players={players}
          invalid={invalidCompositionIds.has(composition.id)}
        />
      ))}
    </div>
  );
}

export function GameBoardTable({
  tableCompositions,
  newCompositions,
  players,
  turnActivity,
  canCompose,
  viewerPlayerId,
  showDraftTotal,
  invalidCompositionIds = new Set<string>(),
  invalidEntryKeys = new Set<string>(),
  showNewCompositionDropCue = false,
}: {
  tableCompositions: TableCompositionView[];
  newCompositions: DraftedCompositionView[];
  players: PlayerSnapshot[];
  turnActivity?: TurnActivitySnapshot;
  canCompose: boolean;
  viewerPlayerId?: string;
  showDraftTotal: boolean;
  invalidCompositionIds?: Set<string>;
  invalidEntryKeys?: Set<string>;
  showNewCompositionDropCue?: boolean;
}) {
  const { active } = useDndContext();
  const shouldReduceMotion = useShouldReduceMotion();
  const draftSectionRef = useRef<HTMLDivElement>(null);
  const previousNewCompositionCountRef = useRef(newCompositions.length);
  const { setNodeRef, isOver: isOverNewCompositionBoard } = useDroppable({
    id: NEW_COMPOSITION_DROP_ID,
    disabled: !canCompose,
  });
  const activityByIndex = new Map<number, CompositionActivitySnapshot>(
    (turnActivity?.compositionActivities ?? []).map((activity: CompositionActivitySnapshot) => [
      activity.tableIndex,
      activity,
    ]),
  );
  const stagedDrafts = turnActivity?.draftCompositions ?? [];
  const spectatorDraftsByTableIndex = new Map(
    stagedDrafts.map((draft) => [draft.tableIndex, draft]),
  );
  const stagedNewDrafts = stagedDrafts.filter(
    (composition) => composition.tableIndex === undefined,
  );
  const isDraggingHandCard =
    active !== null &&
    active.data.current?.drawSource === undefined &&
    typeof active.id === "string";

  useEffect(() => {
    const previousCount = previousNewCompositionCountRef.current;
    previousNewCompositionCountRef.current = newCompositions.length;

    if (newCompositions.length <= previousCount) {
      return;
    }

    const isMobileLayout = window.matchMedia
      ? window.matchMedia("(max-width: 79.999rem)").matches
      : window.innerWidth < 1280;
    if (!isMobileLayout) {
      return;
    }

    const draftSection = draftSectionRef.current;
    if (typeof draftSection?.scrollIntoView !== "function") {
      return;
    }

    draftSection.scrollIntoView({
      behavior: shouldReduceMotion ? "auto" : "smooth",
      block: "nearest",
      inline: "nearest",
    });
  }, [newCompositions.length, shouldReduceMotion]);

  const visibleTableCompositions: VisibleTableComposition[] = tableCompositions.map(
    (composition) => {
      const remoteDraft = spectatorDraftsByTableIndex.get(composition.tableIndex);
      return {
        ...composition,
        interactive: composition.draft !== undefined,
        draftPlayerId: remoteDraft ? turnActivity?.playerId : undefined,
        draft:
          composition.draft ??
          (remoteDraft ? draftPreviewForComposition(composition.snapshot, remoteDraft) : undefined),
      };
    },
  );

  const additionPointTotals = visibleTableCompositions.flatMap(({ snapshot, draft }) => {
    if (!draft || (draft.additions.length === 0 && draft.reclaims.length === 0)) return [];

    const previewPoints = draftCompositionPointTotal(
      cardsWithTableDraft(snapshot, draft),
      snapshot.type,
    );
    const reclaimPoints = draft.reclaims.reduce<number | null>((total, reclaim) => {
      if (total === null) return null;

      const points = jokerReclaimPointValue(
        snapshot,
        reclaim.jokerIndex,
        reclaim.replacementEntry.card,
      );
      return points === null ? null : total + points;
    }, 0);

    return [
      previewPoints === null || reclaimPoints === null
        ? null
        : Math.max(0, previewPoints - snapshot.points) + reclaimPoints,
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
    <Card
      ref={setNodeRef}
      data-onboarding-target="table"
      data-over={isOverNewCompositionBoard || undefined}
      className={cn(
        "h-full min-h-0 max-w-full min-w-0 overflow-hidden transition-colors [--card-spacing:--spacing(2)] xl:flex-1 xl:[--card-spacing:--spacing(6)]",
        canCompose && isDraggingHandCard && isOverNewCompositionBoard
          ? "bg-primary/5 ring-2 ring-primary/70"
          : null,
      )}
    >
      <CardContent className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-auto overscroll-contain px-2 py-2 xl:px-3 xl:py-3">
        <div className="my-auto flex w-full shrink-0 flex-col gap-3 xl:gap-6">
          <TableCompositionsSection
            tableCompositions={visibleTableCompositions}
            activityByIndex={activityByIndex}
            players={players}
            canCompose={canCompose}
            viewerPlayerId={viewerPlayerId}
            invalidEntryKeys={invalidEntryKeys}
          />

          <DraftCompositionsSection
            sectionRef={draftSectionRef}
            showDraftTotal={showDraftTotal}
            visibleDraftPointTotals={visibleDraftPointTotals}
            visibleDraftPointsTotal={visibleDraftPointsTotal}
            stagedNewDrafts={stagedNewDrafts}
            newCompositions={newCompositions}
            players={players}
            turnPlayerId={turnActivity?.playerId}
            animateStagedDrafts={Boolean(
              turnActivity?.playerId && turnActivity.playerId !== viewerPlayerId,
            )}
            invalidCompositionIds={invalidCompositionIds}
            showNewCompositionDropCue={showNewCompositionDropCue}
          />
        </div>
      </CardContent>
    </Card>
  );
}

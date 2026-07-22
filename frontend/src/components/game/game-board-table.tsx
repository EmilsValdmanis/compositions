import { useEffect, useRef, type RefObject } from "react";
import { useDndContext, useDroppable } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  NEW_COMPOSITION_DROP_ID,
  type DraftedCompositionView,
  type TableCompositionView,
  draftCompositionDropId,
} from "#/components/game/game-board-view-state";
import { draftPreviewForComposition } from "#/components/game/game-board-table-state";
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
import { useShouldReduceMotion } from "#/lib/reduced-motion";
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

type SpectatorDraftPreview = ReturnType<typeof draftPreviewForComposition> & {
  playerId?: string;
};

function TableComposition({
  composition,
  spectatorDraft,
  players,
  activity,
  canCompose,
  invalidEntryKeys,
}: {
  composition: TableCompositionView;
  spectatorDraft?: SpectatorDraftPreview;
  players: PlayerSnapshot[];
  activity?: CompositionActivitySnapshot;
  canCompose: boolean;
  invalidEntryKeys: Set<string>;
}) {
  const hasInteractiveEntries = composition.stagedEntries.length > 0;
  const stagedEntries = hasInteractiveEntries
    ? composition.stagedEntries
    : (spectatorDraft?.stagedEntries ?? []);
  const reclaims = hasInteractiveEntries ? composition.reclaims : (spectatorDraft?.reclaims ?? []);
  const insertIndex = hasInteractiveEntries ? composition.insertIndex : spectatorDraft?.insertIndex;
  const cardInsertIndices = hasInteractiveEntries
    ? composition.cardInsertIndices
    : spectatorDraft?.cardInsertIndices;

  return (
    <div className="w-fit shrink-0 overflow-visible p-1">
      <CompositionRow
        composition={composition.snapshot}
        index={composition.tableIndex}
        stagedEntries={stagedEntries}
        reclaims={reclaims}
        insertIndex={insertIndex}
        cardInsertIndices={cardInsertIndices}
        players={players}
        stagedEntryPlayerId={spectatorDraft?.playerId}
        stagedEntriesInteractive={hasInteractiveEntries}
        dropTargetsEnabled={canCompose}
        activity={activity}
        invalidEntryKeys={invalidEntryKeys}
      />
    </div>
  );
}

function TableCompositionsSection({
  tableCompositions,
  spectatorDraftsByTableIndex,
  activityByIndex,
  players,
  canCompose,
  invalidEntryKeys,
}: {
  tableCompositions: TableCompositionView[];
  spectatorDraftsByTableIndex: Map<number, SpectatorDraftPreview>;
  activityByIndex: Map<number, CompositionActivitySnapshot>;
  players: PlayerSnapshot[];
  canCompose: boolean;
  invalidEntryKeys: Set<string>;
}) {
  if (tableCompositions.length === 0) {
    return (
      <div className="w-full shrink-0 overflow-visible">
        <Caption className="mx-auto grid min-h-32 w-fit max-w-full place-items-center rounded-3xl border border-dashed border-border/70 px-4 text-center">
          {m.no_compositions()}
        </Caption>
      </div>
    );
  }

  return (
    <div className="w-full shrink-0 overflow-visible">
      <div className="flex w-full flex-wrap items-center justify-center gap-2 overflow-visible p-1 xl:gap-4">
        {tableCompositions.map((composition) => (
          <TableComposition
            key={composition.key}
            composition={composition}
            spectatorDraft={spectatorDraftsByTableIndex.get(composition.tableIndex)}
            players={players}
            activity={activityByIndex.get(composition.tableIndex)}
            canCompose={canCompose}
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
}: {
  composition: DraftCompositionSnapshot;
  players: PlayerSnapshot[];
  playerId?: string;
}) {
  const pointTotal = draftCompositionPointTotal(composition.cards);

  return (
    <div className="flex w-fit shrink-0 flex-col rounded-3xl border border-primary/70 bg-primary/5 p-3">
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
      <div className="flex items-start gap-2">
        {draftCardInstances(composition.cards).map(({ card, key }) => (
          <GameCard key={`${composition.tableIndex ?? "new"}-${key}`} card={card} size="default" />
        ))}
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

  return (
    <GameBoardDraftDropZone
      id={draftCompositionDropId(composition.id)}
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
  invalidCompositionIds,
}: {
  sectionRef: RefObject<HTMLDivElement | null>;
  showDraftTotal: boolean;
  visibleDraftPointTotals: (number | null)[];
  visibleDraftPointsTotal: number | null;
  stagedNewDrafts: DraftCompositionSnapshot[];
  newCompositions: DraftedCompositionView[];
  players: PlayerSnapshot[];
  turnPlayerId?: string;
  invalidCompositionIds: Set<string>;
}) {
  return (
    <div
      ref={sectionRef}
      className="flex w-full shrink-0 flex-wrap items-center justify-center gap-2 xl:gap-3"
    >
      {showDraftTotal && visibleDraftPointTotals.length > 0 ? (
        <DraftTotal points={visibleDraftPointsTotal} />
      ) : null}

      {stagedNewDrafts.map((composition) => (
        <SpectatorNewDraft
          key={composition.id}
          composition={composition}
          players={players}
          playerId={turnPlayerId}
        />
      ))}

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
  const tableCompositionsByIndex = new Map(
    tableCompositions.map((composition) => [composition.tableIndex, composition]),
  );
  const spectatorDraftsByTableIndex = new Map<number, SpectatorDraftPreview>();
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
    <Card
      ref={setNodeRef}
      data-over={isOverNewCompositionBoard || undefined}
      className={cn(
        "h-full min-h-0 min-w-0 max-w-full overflow-hidden transition-colors [--card-spacing:--spacing(2)] xl:flex-1 xl:[--card-spacing:--spacing(6)]",
        canCompose && isDraggingHandCard && isOverNewCompositionBoard
          ? "bg-primary/5 ring-2 ring-primary/70"
          : null,
      )}
    >
      <CardContent className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-auto overscroll-contain px-2 py-2 xl:px-3 xl:py-3">
        <div className="my-auto flex w-full shrink-0 flex-col gap-3 xl:gap-6">
          <TableCompositionsSection
            tableCompositions={tableCompositions}
            spectatorDraftsByTableIndex={spectatorDraftsByTableIndex}
            activityByIndex={activityByIndex}
            players={players}
            canCompose={canCompose}
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
            invalidCompositionIds={invalidCompositionIds}
          />
        </div>
      </CardContent>
    </Card>
  );
}

import { useEffect, useEffectEvent, useMemo, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type TablePlayRequest,
  type TurnActivitySnapshot,
  useGameWebSocket,
} from "#/components/game-websocket-provider";
import { GameBoardHand } from "#/components/game/game-board-hand";
import { GameBoardPiles } from "#/components/game/game-board-piles";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { GameBoardTable } from "#/components/game/game-board-table";
import { GameCard } from "#/components/game/game-card";
import {
  setPersistedHandOrder,
  usePersistedHandOrder,
} from "#/components/game/game-hand-order-store";
import {
  FACE_DOWN_CARD,
  HAND_DROP_ID,
  NEW_COMPOSITION_DROP_ID,
  type ActiveDrag,
  type DraftComposition,
  type DraftedCompositionView,
  applyHandEntryOrder,
  buildTablePlayRequest,
  buildTableCompositionViews,
  buildVirtualReclaimedJokers,
  buildHandEntries,
  compositionIdFromDropId,
  findNewHandEntry,
  handEntryOrder,
  insertHandKeyIntoDraft,
  mapHandKeysToEntries,
  moveHandEntry,
  moveDraftCompositionInsertIndex,
  pruneDraftCompositions,
  removeHandKeyFromDrafts,
  setDraftCardInsertIndex,
  tableCompositionEdgeTargetFromDropId,
  tableCompositionInsertIndexForEdge,
  tableCompositionIndexFromDropId,
} from "#/components/game/game-board-view-state";
import { Button } from "#/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog";

type GameBoardTurnState = {
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
};

type HandOrderState = {
  scopeKey: string | null;
  order: string[] | null;
};

type DraftCompositionState = {
  scopeKey: string | null;
  compositions: DraftComposition[];
};

type GameBoardViewProps = {
  game: GameSnapshot | null;
  roomCode: string | null;
  playerId: string;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  onDiscardCard: (cardIndex: number) => void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
  onPlayTable: (play: TablePlayRequest) => void;
  disableDraftSync?: boolean;
};

type SubmittedCardActivity = {
  kind: "new" | "addition" | "joker_reclaim";
  playerId: string;
};

type SubmittedCompositionActivity = {
  kind?: string;
  playerId?: string;
  cardActivities: Record<number, SubmittedCardActivity>;
};

function draftScopeKey(game: GameSnapshot | null, playerId: string, isMyTurn: boolean) {
  if (!game || !isMyTurn) {
    return null;
  }

  return `${playerId}:${game.round}:${game.turn.number}`;
}

function buildDraftedCompositionViews(
  draftCompositions: DraftComposition[],
  entryByKey: Map<string, ReturnType<typeof buildHandEntries>[number]>,
) {
  return draftCompositions.map((composition) => ({
    ...composition,
    entries: mapHandKeysToEntries(composition.handKeys, entryByKey),
  })) as DraftedCompositionView[];
}

function cardEquals(a: GameSnapshot["hand"][number], b: GameSnapshot["hand"][number]) {
  return Boolean(a.isJoker) === Boolean(b.isJoker) && a.rank === b.rank && a.suit === b.suit;
}

function buildSubmittedCompositionActivityMap(
  baselineCompositions: GameSnapshot["activeCompositions"],
  currentCompositions: GameSnapshot["activeCompositions"],
  turnActivity?: TurnActivitySnapshot,
) {
  const activityMap = new Map<number, SubmittedCompositionActivity>();
  const compositionActivityByIndex = new Map(
    (turnActivity?.compositionActivities ?? []).map((compositionActivity) => [
      compositionActivity.tableIndex,
      compositionActivity,
    ]),
  );

  for (const compositionActivity of turnActivity?.compositionActivities ?? []) {
    activityMap.set(compositionActivity.tableIndex, {
      kind: compositionActivity.kind,
      playerId: compositionActivity.playerId,
      cardActivities: {},
    });
  }

  for (let index = 0; index < currentCompositions.length; index += 1) {
    const current = currentCompositions[index];
    const baseline = baselineCompositions[index];
    const activity = activityMap.get(index) ?? {
      kind: undefined,
      playerId: undefined,
      cardActivities: {},
    };

    if (!baseline) {
      activity.kind ??= "new_composition";
      for (let cardIndex = 0; cardIndex < current.cards.length; cardIndex += 1) {
        const activityDetails = compositionActivityByIndex.get(index)?.cardActivities?.[cardIndex];
        if (activityDetails) {
          activity.cardActivities[cardIndex] = {
            kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "new",
            playerId: activityDetails.playerId,
          };
        }
      }
      activityMap.set(index, activity);
      continue;
    }

    for (let cardIndex = 0; cardIndex < current.cards.length; cardIndex += 1) {
      const currentCard = current.cards[cardIndex];
      const baselineCard = baseline.cards[cardIndex];
      const activityDetails = compositionActivityByIndex.get(index)?.cardActivities?.[cardIndex];

      if (!baselineCard) {
        if (activityDetails) {
          activity.cardActivities[cardIndex] = {
            kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "addition",
            playerId: activityDetails.playerId,
          };
          activityMap.set(index, activity);
        }
        continue;
      }

      if (
        activityDetails &&
        (activityDetails.kind === "joker_reclaim" || !cardEquals(currentCard, baselineCard))
      ) {
        activity.cardActivities[cardIndex] = {
          kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "addition",
          playerId: activityDetails.playerId,
        };
        activityMap.set(index, activity);
      }
    }
  }

  return activityMap;
}

function GameBoardLayout({
  game,
  tableCompositions,
  newCompositions,
  submittedCompositionActivities,
  turnState,
  topDiscardCard,
  players,
  connectedPlayers,
  availableHandEntries,
  sortableIds,
  activeDrag,
  draftedCompositionsCount,
  spectatorDrafts,
  canSubmitTablePlay,
  onResetTablePlay,
  onSubmitTablePlay,
}: {
  game: GameSnapshot | null;
  tableCompositions: ReturnType<typeof buildTableCompositionViews>;
  newCompositions: DraftedCompositionView[];
  submittedCompositionActivities: Map<number, SubmittedCompositionActivity>;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  availableHandEntries: ReturnType<typeof buildHandEntries>;
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  draftedCompositionsCount: number;
  spectatorDrafts: DraftCompositionSnapshot[];
  canSubmitTablePlay: boolean;
  onResetTablePlay: () => void;
  onSubmitTablePlay: () => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
      <div className="grid min-h-0 flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <GameBoardTable
          game={game}
          tableCompositions={tableCompositions}
          newCompositions={newCompositions}
          players={players}
          turnActivity={
            game?.turnActivity
              ? {
                  ...game.turnActivity,
                  draftCompositions: spectatorDrafts,
                  compositionActivities: Array.from(submittedCompositionActivities.entries()).map(
                    ([tableIndex, activity]) => ({
                      tableIndex,
                      kind: activity.kind,
                      playerId: activity.playerId,
                      cardActivities: activity.cardActivities,
                    }),
                  ),
                }
              : undefined
          }
          canCompose={turnState.canDiscard}
          hasDraftedCompositions={draftedCompositionsCount > 0}
          canSubmitTablePlay={canSubmitTablePlay}
          onResetTablePlay={onResetTablePlay}
          onSubmitTablePlay={onSubmitTablePlay}
        />

        <div className="flex flex-col min-h-0 gap-4">
          <GameBoardPlayers players={players} game={game} connectedPlayers={connectedPlayers} />
          <GameBoardPiles
            drawPileCount={game?.drawPileCount ?? 0}
            topDiscardCard={topDiscardCard}
            canDrawDeck={turnState.canDrawDeck}
            canDrawDiscard={turnState.canDrawDiscard}
            canDiscard={turnState.canDiscard}
          />
        </div>
      </div>

      <div className="grid min-h-0 gap-4">
        <GameBoardHand
          status={{
            hasGame: Boolean(game),
            isMyTurn: turnState.isMyTurn,
            turnPlayerName: turnState.turnPlayerName,
          }}
          availableHandEntries={availableHandEntries}
          sortableIds={sortableIds}
          activeDrag={activeDrag}
          tablePlayState={{
            hasDraftedCompositions: draftedCompositionsCount > 0,
          }}
        />
      </div>
    </div>
  );
}

type GameBoardController = {
  activeDrag: ActiveDrag | null;
  activeEntry: ReturnType<typeof buildHandEntries>[number] | null;
  availableHandEntries: ReturnType<typeof buildHandEntries>;
  sortableIds: string[];
  tableCompositions: ReturnType<typeof buildTableCompositionViews>;
  newCompositions: DraftedCompositionView[];
  draftedCompositionsCount: number;
  canSubmitTablePlay: boolean;
  pendingDiscardIndex: number | null;
  handleDragStart: (event: DragStartEvent) => void;
  handleDragOver: (event: DragOverEvent) => void;
  handleDragEnd: (event: DragEndEvent) => void;
  handleDragCancel: () => void;
  closePendingDiscardDialog: () => void;
  discardWithoutSubmitting: () => void;
  submitBeforeDiscard: () => void;
  resetDraftCompositions: () => void;
  submitDraftCompositions: () => void;
};

function useGameBoardController({
  game,
  roomCode,
  playerId,
  turnState,
  topDiscardCard,
  onDiscardCard,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
  disableDraftSync = false,
}: Pick<
  GameBoardViewProps,
  | "game"
  | "roomCode"
  | "playerId"
  | "turnState"
  | "topDiscardCard"
  | "onDiscardCard"
  | "onDrawFromDeck"
  | "onDrawFromDiscard"
  | "onPlayTable"
  | "disableDraftSync"
>): GameBoardController {
  const { updateTurnDrafts } = useGameWebSocket();
  const handOrderScopeKey = useMemo(
    () => (roomCode && playerId ? `${roomCode}:${playerId}` : null),
    [playerId, roomCode],
  );
  const persistedHandOrder = usePersistedHandOrder(handOrderScopeKey);
  const [handOrderState, setHandOrderState] = useState<HandOrderState>({
    scopeKey: null,
    order: null,
  });
  const [activeDrag, setActiveDrag] = useState<ActiveDrag | null>(null);
  const [draftCompositionState, setDraftCompositionState] = useState<DraftCompositionState>({
    scopeKey: null,
    compositions: [],
  });
  const [pendingDiscardIndex, setPendingDiscardIndex] = useState<number | null>(null);
  const nextDraftIdRef = useRef(0);
  const rawHandEntries = useMemo(() => buildHandEntries(game?.hand ?? []), [game?.hand]);
  const handOrder =
    handOrderState.scopeKey === handOrderScopeKey && handOrderState.order
      ? handOrderState.order
      : persistedHandOrder;
  const handEntries = useMemo(
    () => applyHandEntryOrder(rawHandEntries, handOrder),
    [handOrder, rawHandEntries],
  );
  const currentDraftScopeKey = draftScopeKey(game, playerId, turnState.isMyTurn);

  function updateHandOrder(order: string[]) {
    setHandOrderState({
      scopeKey: handOrderScopeKey,
      order,
    });
    setPersistedHandOrder(handOrderScopeKey, order);
  }

  const activeDrawEntry =
    activeDrag?.type === "draw"
      ? findNewHandEntry(
          activeDrag.baselineEntries,
          applyHandEntryOrder(rawHandEntries, activeDrag.baselineOrder),
        )
      : null;

  function updateDraftCompositions(updater: (current: DraftComposition[]) => DraftComposition[]) {
    setDraftCompositionState((current) => {
      const scopedCompositions =
        current.scopeKey === currentDraftScopeKey ? current.compositions : [];

      return {
        scopeKey: currentDraftScopeKey,
        compositions: updater(scopedCompositions),
      };
    });
  }

  const validHandKeys = useMemo(
    () => new Set(handEntries.map((entry) => entry.key)),
    [handEntries],
  );
  const scopedDraftCompositions = useMemo(
    () =>
      draftCompositionState.scopeKey === currentDraftScopeKey
        ? draftCompositionState.compositions
        : [],
    [currentDraftScopeKey, draftCompositionState.compositions, draftCompositionState.scopeKey],
  );
  const draftCompositionsFromHand = useMemo(
    () => pruneDraftCompositions(scopedDraftCompositions, validHandKeys),
    [scopedDraftCompositions, validHandKeys],
  );
  const entryByKey = useMemo(
    () => new Map(handEntries.map((entry) => [entry.key, entry])),
    [handEntries],
  );
  const draftedCompositionsViewFromHand = useMemo(
    () => buildDraftedCompositionViews(draftCompositionsFromHand, entryByKey),
    [draftCompositionsFromHand, entryByKey],
  );
  const tableCompositions = useMemo(
    () =>
      buildTableCompositionViews(
        game?.activeCompositions ?? [],
        draftedCompositionsViewFromHand,
        entryByKey,
      ),
    [draftedCompositionsViewFromHand, entryByKey, game?.activeCompositions],
  );
  const virtualReclaimedJokers = useMemo(
    () => buildVirtualReclaimedJokers(tableCompositions),
    [tableCompositions],
  );
  const allHandEntries = useMemo(
    () => [...handEntries, ...virtualReclaimedJokers.map((joker) => joker.entry)],
    [handEntries, virtualReclaimedJokers],
  );
  const allValidHandKeys = useMemo(
    () => new Set(allHandEntries.map((entry) => entry.key)),
    [allHandEntries],
  );
  const draftCompositions = useMemo(
    () => pruneDraftCompositions(scopedDraftCompositions, allValidHandKeys),
    [allValidHandKeys, scopedDraftCompositions],
  );
  const activeEntry =
    activeDrag?.type === "hand"
      ? (allHandEntries.find((entry) => entry.key === activeDrag.handKey) ?? null)
      : null;
  const allEntryByKey = useMemo(
    () => new Map(allHandEntries.map((entry) => [entry.key, entry])),
    [allHandEntries],
  );
  const stagedHandKeySet = useMemo(
    () => new Set(draftCompositions.flatMap((composition) => composition.handKeys)),
    [draftCompositions],
  );
  const availableHandEntries = useMemo(
    () => allHandEntries.filter((entry) => !stagedHandKeySet.has(entry.key)),
    [allHandEntries, stagedHandKeySet],
  );
  const sortableIds = useMemo(
    () => availableHandEntries.map((entry) => entry.key),
    [availableHandEntries],
  );
  const draftedCompositionsView = useMemo(
    () => buildDraftedCompositionViews(draftCompositions, allEntryByKey),
    [draftCompositions, allEntryByKey],
  );
  const newCompositions = useMemo(
    () => draftedCompositionsView.filter((composition) => composition.tableIndex === null),
    [draftedCompositionsView],
  );
  const canCompose = turnState.canDiscard;
  const canSubmitTablePlay = canCompose && draftedCompositionsView.length > 0;

  const serializedDrafts = useMemo(
    () =>
      JSON.stringify(
        draftedCompositionsView.map(
          (composition) =>
            ({
              tableIndex: composition.tableIndex ?? undefined,
              insertIndex: composition.insertIndex,
              cards: composition.entries.map((entry) => entry.card),
            }) satisfies DraftCompositionSnapshot,
        ),
      ),
    [draftedCompositionsView],
  );
  const syncTurnDrafts = useEffectEvent((drafts: DraftCompositionSnapshot[]) => {
    updateTurnDrafts({ compositions: drafts });
  });

  useEffect(() => {
    if (disableDraftSync || !turnState.isMyTurn) {
      return;
    }

    syncTurnDrafts(JSON.parse(serializedDrafts) as DraftCompositionSnapshot[]);
  }, [disableDraftSync, serializedDrafts, turnState.isMyTurn]);

  function resetDraftCompositions() {
    setDraftCompositionState({
      scopeKey: currentDraftScopeKey,
      compositions: [],
    });
  }

  function submitDraftCompositions() {
    if (!canSubmitTablePlay) {
      return;
    }

    onPlayTable(buildTablePlayRequest(game?.activeCompositions ?? [], draftedCompositionsView));
  }

  function closePendingDiscardDialog() {
    setPendingDiscardIndex(null);
  }

  function discardWithoutSubmitting() {
    if (pendingDiscardIndex === null) {
      return;
    }

    resetDraftCompositions();
    onDiscardCard(pendingDiscardIndex);
    closePendingDiscardDialog();
  }

  function submitBeforeDiscard() {
    submitDraftCompositions();
    closePendingDiscardDialog();
  }

  function handleDragStart(event: DragStartEvent) {
    const drawSource = event.active.data.current?.drawSource;

    if (drawSource === "deck") {
      setActiveDrag({
        type: "draw",
        source: "deck",
        card: null,
        baselineEntries: handEntries,
        baselineOrder: handEntryOrder(handEntries),
        revealedHandKey: null,
      });
      onDrawFromDeck();
      return;
    }

    if (drawSource === "discard" && topDiscardCard) {
      setActiveDrag({
        type: "draw",
        source: "discard",
        card: topDiscardCard,
        baselineEntries: handEntries,
        baselineOrder: handEntryOrder(handEntries),
        revealedHandKey: null,
      });
      onDrawFromDiscard();
      return;
    }

    const handKey = typeof event.active.id === "string" ? event.active.id : null;
    setActiveDrag(
      handKey
        ? {
            type: "hand",
            handKey,
            baselineDraftCompositions: draftCompositions,
          }
        : null,
    );
  }

  function handleDragOver(event: DragOverEvent) {
    if (activeDrag?.type === "draw") {
      if (activeDrawEntry === null) {
        return;
      }

      const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

      if (overHandKey === null || overHandKey === "discard-pile") {
        updateHandOrder(activeDrag.baselineOrder);
        return;
      }

      updateHandOrder(handEntryOrder(moveHandEntry(handEntries, activeDrawEntry.key, overHandKey)));
      return;
    }

    if (activeDrag?.type !== "hand" || !canCompose) {
      return;
    }

    const overId = typeof event.over?.id === "string" ? event.over.id : null;
    if (overId === null) {
      return;
    }

    const droppedOnDraftCard = draftCompositions.find((composition) =>
      composition.handKeys.includes(overId),
    );
    if (droppedOnDraftCard?.tableIndex === null) {
      updateDraftCompositions((current) =>
        insertHandKeyIntoDraft(current, activeDrag.handKey, droppedOnDraftCard.id, overId),
      );
      return;
    }

    const droppedOnDraftContainer = compositionIdFromDropId(overId);
    if (!droppedOnDraftContainer) {
      return;
    }

    const target = draftCompositions.find(
      (composition) => composition.id === droppedOnDraftContainer,
    );
    if (target?.tableIndex !== null) {
      return;
    }

    updateDraftCompositions((current) =>
      insertHandKeyIntoDraft(current, activeDrag.handKey, droppedOnDraftContainer),
    );
  }

  function handleDragEnd(event: DragEndEvent) {
    const currentDrag = activeDrag;
    const draggedHandKey = typeof event.active.id === "string" ? event.active.id : null;
    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;
    const overId = typeof event.over?.id === "string" ? event.over.id : null;

    setActiveDrag(null);

    if (currentDrag?.type === "draw") {
      if (activeDrawEntry?.key && overHandKey && overHandKey !== "discard-pile") {
        return;
      }

      updateHandOrder(currentDrag.baselineOrder);
      return;
    }

    if (currentDrag?.type !== "hand" || typeof draggedHandKey !== "string") {
      return;
    }

    if (overId === null) {
      setDraftCompositionState({
        scopeKey: currentDraftScopeKey,
        compositions: currentDrag.baselineDraftCompositions,
      });
      return;
    }

    const draggedEntry = allEntryByKey.get(draggedHandKey) ?? null;
    const droppedOnDraftCard =
      overId === null
        ? null
        : draftCompositions.find((composition) => composition.handKeys.includes(overId));
    const droppedOnHandCard =
      overId !== null && availableHandEntries.some((entry) => entry.key === overId);
    const droppedOnDraftContainer = overId ? compositionIdFromDropId(overId) : null;
    const droppedOnTableComposition = overId ? tableCompositionIndexFromDropId(overId) : null;
    const droppedOnTableEdgeTarget = overId ? tableCompositionEdgeTargetFromDropId(overId) : null;

    if (event.over?.id === "discard-pile") {
      if (draftCompositions.some((composition) => composition.handKeys.includes(draggedHandKey))) {
        return;
      }

      const cardIndex = event.active.data.current?.cardIndex;
      const isVirtual = event.active.data.current?.isVirtual === true;
      if (isVirtual) {
        return;
      }
      if (typeof cardIndex === "number") {
        if (draftedCompositionsView.length > 0) {
          setPendingDiscardIndex(cardIndex);
        } else {
          onDiscardCard(cardIndex);
        }
      }
      return;
    }

    if (!canCompose) {
      if (droppedOnHandCard && draggedHandKey !== overId) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, draggedHandKey, overId)));
      }
      return;
    }

    if (droppedOnTableEdgeTarget !== null && draggedEntry) {
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableEdgeTarget.compositionIndex,
        );
        const targetComposition =
          game?.activeCompositions?.[droppedOnTableEdgeTarget.compositionIndex];
        const insertIndex = targetComposition
          ? tableCompositionInsertIndexForEdge(targetComposition, droppedOnTableEdgeTarget.edge)
          : 0;

        if (existing) {
          return setDraftCardInsertIndex(
            insertHandKeyIntoDraft(current, draggedHandKey, existing.id),
            existing.id,
            draggedHandKey,
            insertIndex,
          );
        }

        const compositionId = `draft-${nextDraftIdRef.current}`;
        nextDraftIdRef.current += 1;

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          {
            id: compositionId,
            handKeys: [draggedHandKey],
            tableIndex: droppedOnTableEdgeTarget.compositionIndex,
            insertIndex: targetComposition?.cards.length ?? 0,
            cardInsertIndices: { [draggedHandKey]: insertIndex },
          },
        ];
      });
      return;
    }

    if (droppedOnTableComposition !== null && draggedEntry) {
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableComposition,
        );
        if (existing) {
          return moveDraftCompositionInsertIndex(
            insertHandKeyIntoDraft(current, draggedHandKey, existing.id),
            existing.id,
            existing.insertIndex ?? existing.handKeys.length,
          );
        }

        const compositionId = `draft-${nextDraftIdRef.current}`;
        nextDraftIdRef.current += 1;

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          {
            id: compositionId,
            handKeys: [draggedHandKey],
            tableIndex: droppedOnTableComposition,
            insertIndex: game?.activeCompositions?.[droppedOnTableComposition]?.cards.length,
          },
        ];
      });
      return;
    }

    if (overId === NEW_COMPOSITION_DROP_ID) {
      const compositionId = `draft-${nextDraftIdRef.current}`;
      nextDraftIdRef.current += 1;

      updateDraftCompositions((current) => [
        ...removeHandKeyFromDrafts(current, draggedHandKey),
        { id: compositionId, handKeys: [draggedHandKey], tableIndex: null },
      ]);
      return;
    }

    if (droppedOnDraftCard) {
      updateDraftCompositions((current) =>
        insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftCard.id, overId ?? undefined),
      );
      return;
    }

    if (droppedOnDraftContainer) {
      updateDraftCompositions((current) => {
        const target = current.find((composition) => composition.id === droppedOnDraftContainer);
        const next = insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftContainer);

        if (target && target.tableIndex !== null) {
          return moveDraftCompositionInsertIndex(
            next,
            droppedOnDraftContainer,
            target.insertIndex ?? target.handKeys.length,
          );
        }

        return next;
      });
      return;
    }

    if (overId === HAND_DROP_ID) {
      updateDraftCompositions((current) => removeHandKeyFromDrafts(current, draggedHandKey));
      return;
    }

    if (droppedOnHandCard) {
      updateDraftCompositions((current) => removeHandKeyFromDrafts(current, draggedHandKey));

      if (draggedHandKey !== overId) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, draggedHandKey, overId)));
      }
    }
  }

  function handleDragCancel() {
    if (activeDrag?.type === "draw") {
      updateHandOrder(activeDrag.baselineOrder);
    } else if (activeDrag?.type === "hand") {
      setDraftCompositionState({
        scopeKey: currentDraftScopeKey,
        compositions: activeDrag.baselineDraftCompositions,
      });
    }

    setActiveDrag(null);
  }

  const displayActiveDrag =
    activeDrag?.type === "draw"
      ? { ...activeDrag, revealedHandKey: activeDrawEntry?.key ?? null }
      : activeDrag;

  return {
    activeDrag: displayActiveDrag,
    activeEntry,
    availableHandEntries,
    sortableIds,
    tableCompositions,
    newCompositions,
    draftedCompositionsCount: draftedCompositionsView.length,
    canSubmitTablePlay,
    pendingDiscardIndex,
    handleDragStart,
    handleDragOver,
    handleDragEnd,
    handleDragCancel,
    closePendingDiscardDialog,
    discardWithoutSubmitting,
    submitBeforeDiscard,
    resetDraftCompositions,
    submitDraftCompositions,
  };
}

export function GameBoardView({
  game,
  roomCode,
  playerId,
  players,
  connectedPlayers,
  turnState,
  topDiscardCard,
  onDiscardCard,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
  disableDraftSync,
}: GameBoardViewProps) {
  const controller = useGameBoardController({
    game,
    roomCode,
    playerId,
    turnState,
    topDiscardCard,
    onDiscardCard,
    onDrawFromDeck,
    onDrawFromDiscard,
    onPlayTable,
    disableDraftSync,
  });
  const activeDraw = controller.activeDrag?.type === "draw" ? controller.activeDrag : null;
  const submittedCompositionActivities = useMemo(
    () =>
      buildSubmittedCompositionActivityMap(
        game?.turnActivity?.baselineCompositions ?? game?.activeCompositions ?? [],
        game?.activeCompositions ?? [],
        game?.turnActivity,
      ),
    [game?.activeCompositions, game?.turnActivity],
  );
  const spectatorDrafts = game?.turnActivity?.draftCompositions ?? [];

  return (
    <>
      <DndContext
        collisionDetection={closestCenter}
        onDragStart={controller.handleDragStart}
        onDragOver={controller.handleDragOver}
        onDragEnd={controller.handleDragEnd}
        onDragCancel={controller.handleDragCancel}
      >
        <GameBoardLayout
          game={game}
          tableCompositions={controller.tableCompositions}
          newCompositions={controller.newCompositions}
          submittedCompositionActivities={submittedCompositionActivities}
          turnState={turnState}
          topDiscardCard={topDiscardCard}
          players={players}
          connectedPlayers={connectedPlayers}
          availableHandEntries={controller.availableHandEntries}
          sortableIds={controller.sortableIds}
          activeDrag={controller.activeDrag}
          draftedCompositionsCount={controller.draftedCompositionsCount}
          spectatorDrafts={spectatorDrafts}
          canSubmitTablePlay={controller.canSubmitTablePlay}
          onResetTablePlay={controller.resetDraftCompositions}
          onSubmitTablePlay={controller.submitDraftCompositions}
        />
        <DragOverlay dropAnimation={null}>
          {activeDraw ? (
            <GameCard
              card={
                (activeDraw.revealedHandKey
                  ? controller.availableHandEntries.find(
                      (entry) => entry.key === activeDraw.revealedHandKey,
                    )?.card
                  : null) ??
                activeDraw.card ??
                FACE_DOWN_CARD
              }
              faceDown={activeDraw.revealedHandKey === null && activeDraw.card === null}
              size="hand"
              className="rotate-3 shadow-xl ring-1 ring-foreground/10"
            />
          ) : controller.activeEntry ? (
            <GameCard
              card={controller.activeEntry.card}
              size="hand"
              className="rotate-3 shadow-xl ring-1 ring-foreground/10"
            />
          ) : null}
        </DragOverlay>
      </DndContext>

      <Dialog
        open={controller.pendingDiscardIndex !== null}
        onOpenChange={(open) => !open && controller.closePendingDiscardDialog()}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Submit staged cards first?</DialogTitle>
            <DialogDescription>
              You have cards staged in new compositions or table additions. Submit them now, or
              discard anyway and clear those staged cards.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="destructive"
              onClick={controller.discardWithoutSubmitting}
            >
              Discard anyway
            </Button>
            <Button
              type="button"
              onClick={controller.submitBeforeDiscard}
              disabled={!controller.canSubmitTablePlay}
            >
              Submit staged cards
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

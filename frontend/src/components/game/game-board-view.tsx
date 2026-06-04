import { useEffect, useEffectEvent, useRef, useState } from "react";
import {
  closestCenter,
  DndContext,
  DragOverlay,
  pointerWithin,
  type CollisionDetection,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  type ActionResult,
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
  buildHandEntries,
  compositionIdFromDropId,
  findNewHandEntry,
  handIndexAfterSubmittedDrafts,
  handEntryOrder,
  insertHandKeyIntoDraft,
  mapHandKeysToEntries,
  moveHandEntry,
  moveDraftCompositionInsertIndex,
  removeHandKeyFromDrafts,
  resolveDraftViews,
  setDraftCardInsertIndex,
  setDraftCardReclaimTarget,
  tableCompositionEdgeTargetFromDropId,
  tableCompositionInsertIndexForEdge,
  tableCompositionJokerTargetFromDropId,
} from "#/components/game/game-board-view-state";

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

function createBoardCollisionDetection(handCardIds: Set<string>): CollisionDetection {
  return ({ droppableContainers, ...args }) => {
    const handCardContainers = droppableContainers.filter((container) =>
      handCardIds.has(String(container.id)),
    );

    const pointerCollisions = pointerWithin({
      ...args,
      droppableContainers,
    });
    const pointerSpecificCollisions = pointerCollisions.filter(
      (collision) => collision.id !== NEW_COMPOSITION_DROP_ID && collision.id !== HAND_DROP_ID,
    );
    const jokerSpecificCollisions = pointerSpecificCollisions.filter((collision) =>
      String(collision.id).startsWith("table-composition-joker-"),
    );
    const edgeSpecificCollisions = pointerSpecificCollisions.filter((collision) =>
      String(collision.id).startsWith("table-composition-edge-"),
    );

    if (jokerSpecificCollisions.length > 0) {
      return jokerSpecificCollisions;
    }

    if (edgeSpecificCollisions.length > 0) {
      return edgeSpecificCollisions;
    }

    if (pointerSpecificCollisions.length > 0) {
      return pointerSpecificCollisions;
    }

    if (
      pointerCollisions.some((collision) => collision.id === HAND_DROP_ID) &&
      handCardContainers.length > 0
    ) {
      const closestHandCollisions = closestCenter({
        ...args,
        droppableContainers: handCardContainers,
      });

      if (closestHandCollisions.length > 0) {
        return closestHandCollisions;
      }
    }

    if (pointerCollisions.length > 0) {
      return pointerCollisions;
    }

    return [];
  };
}

type GameBoardViewProps = {
  game: GameSnapshot | null;
  roomCode: string | null;
  playerId: string;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  onDiscardCard: (cardIndex: number) => Promise<ActionResult> | void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
  onPlayTable: (play: TablePlayRequest) => Promise<ActionResult> | void;
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
  onResetDraftCompositions,
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
  onResetDraftCompositions: () => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="grid min-h-0 flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <GameBoardTable
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
        />

        <div className="flex flex-col min-h-0 gap-4">
          <GameBoardPlayers
            players={players}
            game={game}
            connectedPlayers={connectedPlayers}
            hasDraftedCompositions={draftedCompositionsCount > 0}
            onResetDraftCompositions={onResetDraftCompositions}
          />
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
  handleDragStart: (event: DragStartEvent) => void;
  handleDragOver: (event: DragOverEvent) => void;
  handleDragEnd: (event: DragEndEvent) => void;
  handleDragCancel: () => void;
  resetDraftCompositions: () => void;
  submitDraftCompositions: () => Promise<ActionResult | void>;
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
  const handOrderScopeKey = roomCode && playerId ? `${roomCode}:${playerId}` : null;
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
  const nextDraftIdRef = useRef(0);
  const skipNextDraftSyncRef = useRef(false);
  const rawHandEntries = buildHandEntries(game?.hand ?? []);
  const handOrder =
    handOrderState.scopeKey === handOrderScopeKey && handOrderState.order
      ? handOrderState.order
      : persistedHandOrder;
  const handEntries = applyHandEntryOrder(rawHandEntries, handOrder);
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

  const scopedDraftCompositions =
    draftCompositionState.scopeKey === currentDraftScopeKey
      ? draftCompositionState.compositions
      : [];
  const draftResolution = resolveDraftViews(
    game?.activeCompositions ?? [],
    scopedDraftCompositions,
    handEntries,
  );
  const tableCompositions = draftResolution.tableCompositions;
  const allHandEntries = draftResolution.allHandEntries;
  const draftCompositions = draftResolution.draftCompositions;
  const activeEntry =
    activeDrag?.type === "hand"
      ? (allHandEntries.find((entry) => entry.key === activeDrag.handKey) ?? null)
      : null;
  const allEntryByKey = draftResolution.allEntryByKey;
  const stagedHandKeySet = new Set(
    draftCompositions.flatMap((composition) => composition.handKeys),
  );
  const availableHandEntries = allHandEntries.filter((entry) => !stagedHandKeySet.has(entry.key));
  const sortableIds = availableHandEntries.map((entry) => entry.key);
  const draftedCompositionsView = buildDraftedCompositionViews(draftCompositions, allEntryByKey);
  const newCompositions = draftedCompositionsView.filter(
    (composition) => composition.tableIndex === null,
  );
  const canCompose = turnState.canDiscard;
  const canSubmitTablePlay = canCompose && draftedCompositionsView.length > 0;

  const serializedDrafts = JSON.stringify(
    draftedCompositionsView.map(
      (composition) =>
        ({
          tableIndex: composition.tableIndex ?? undefined,
          insertIndex: composition.insertIndex,
          cardInsertIndices: composition.cardInsertIndices,
          reclaimTargets: composition.reclaimTargets,
          cards: composition.entries.map((entry) => entry.card),
        }) satisfies DraftCompositionSnapshot,
    ),
  );
  const syncTurnDrafts = useEffectEvent((drafts: DraftCompositionSnapshot[]) => {
    updateTurnDrafts({ compositions: drafts });
  });

  useEffect(() => {
    if (disableDraftSync || !turnState.isMyTurn) {
      return;
    }

    if (skipNextDraftSyncRef.current) {
      skipNextDraftSyncRef.current = false;
      return;
    }

    syncTurnDrafts(JSON.parse(serializedDrafts) as DraftCompositionSnapshot[]);
  }, [disableDraftSync, serializedDrafts, turnState.isMyTurn]);

  function resetDraftCompositions({ sync = true }: { sync?: boolean } = {}) {
    if (!sync) {
      skipNextDraftSyncRef.current = true;
    }

    setDraftCompositionState({
      scopeKey: currentDraftScopeKey,
      compositions: [],
    });
  }

  async function submitDraftCompositions() {
    if (!canSubmitTablePlay) {
      return;
    }

    const result = await onPlayTable(
      buildTablePlayRequest(game?.activeCompositions ?? [], draftedCompositionsView),
    );

    resetDraftCompositions({ sync: false });
    return result;
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
            baselineHandOrder: handEntryOrder(availableHandEntries),
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
    const droppedOnTableEdgeTarget = overId ? tableCompositionEdgeTargetFromDropId(overId) : null;
    const droppedOnTableJokerTarget = overId ? tableCompositionJokerTargetFromDropId(overId) : null;

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
        void (async () => {
          const discardIndex =
            draftedCompositionsView.length > 0
              ? handIndexAfterSubmittedDrafts(
                  rawHandEntries,
                  draggedHandKey,
                  draftedCompositionsView,
                )
              : cardIndex;

          if (discardIndex === null) {
            return;
          }

          if (draftedCompositionsView.length > 0) {
            await submitDraftCompositions();
          }

          await onDiscardCard(discardIndex);
        })();
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

    if (droppedOnTableJokerTarget !== null && draggedEntry) {
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableJokerTarget.compositionIndex,
        );

        if (existing) {
          return setDraftCardReclaimTarget(
            insertHandKeyIntoDraft(current, draggedHandKey, existing.id),
            existing.id,
            draggedHandKey,
            droppedOnTableJokerTarget.jokerIndex,
          );
        }

        const compositionId = `draft-${nextDraftIdRef.current}`;
        nextDraftIdRef.current += 1;

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          {
            id: compositionId,
            handKeys: [draggedHandKey],
            tableIndex: droppedOnTableJokerTarget.compositionIndex,
            insertIndex:
              game?.activeCompositions?.[droppedOnTableJokerTarget.compositionIndex]?.cards
                .length ?? 0,
            reclaimTargets: { [draggedHandKey]: droppedOnTableJokerTarget.jokerIndex },
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
    handleDragStart,
    handleDragOver,
    handleDragEnd,
    handleDragCancel,
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
  const hasSubmittedTurnActivity =
    (game?.turnActivity?.compositionActivities?.length ?? 0) > 0;
  const submittedCompositionActivities = hasSubmittedTurnActivity
    ? new Map<number, SubmittedCompositionActivity>()
    : buildSubmittedCompositionActivityMap(
        game?.turnActivity?.baselineCompositions ?? game?.activeCompositions ?? [],
        game?.activeCompositions ?? [],
        game?.turnActivity,
      );
  const spectatorDrafts = hasSubmittedTurnActivity
    ? []
    : (game?.turnActivity?.draftCompositions ?? []);
  const collisionDetection = createBoardCollisionDetection(
    new Set(controller.availableHandEntries.map((entry) => entry.key)),
  );

  return (
    <DndContext
      collisionDetection={collisionDetection}
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
        onResetDraftCompositions={controller.resetDraftCompositions}
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
  );
}

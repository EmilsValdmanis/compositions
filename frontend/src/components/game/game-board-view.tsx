import { useMemo, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  type GameSnapshot,
  type PlayerSnapshot,
  type TablePlayRequest,
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
  buildTableCompositionViews,
  buildHandEntries,
  compositionIdFromDropId,
  findNewHandEntry,
  handEntryOrder,
  insertHandKeyIntoDraft,
  mapHandKeysToEntries,
  moveHandEntry,
  pruneDraftCompositions,
  removeHandKeyFromDrafts,
  tableCompositionIndexFromDropId,
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

type GameBoardViewProps = {
  game: GameSnapshot | null;
  roomCode: string | null;
  playerId: string;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  onDragEnd: (event: DragEndEvent) => void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
  onPlayTable: (play: TablePlayRequest) => void;
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

function buildTablePlayRequest(
  draftedCompositionsView: DraftedCompositionView[],
): TablePlayRequest {
  const compositions: TablePlayRequest["compositions"] = [];
  const additions: TablePlayRequest["additions"] = [];

  for (const composition of draftedCompositionsView) {
    const cards = composition.entries.map((entry) => entry.card);

    if (composition.tableIndex === null) {
      compositions.push({ cards });
      continue;
    }

    additions.push({
      compositionIndex: composition.tableIndex,
      cards,
    });
  }

  return {
    compositions,
    additions,
    reclaims: [],
  };
}

function GameBoardLayout({
  game,
  tableCompositions,
  turnState,
  topDiscardCard,
  players,
  connectedPlayers,
  availableHandEntries,
  sortableIds,
  activeDrag,
  draftedCompositionsCount,
  canSubmitTablePlay,
  onResetTablePlay,
  onSubmitTablePlay,
}: {
  game: GameSnapshot | null;
  tableCompositions: ReturnType<typeof buildTableCompositionViews>;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  availableHandEntries: ReturnType<typeof buildHandEntries>;
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  draftedCompositionsCount: number;
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
          canCompose={turnState.canDiscard}
        />

        <div className="grid min-h-0 auto-rows-fr gap-4">
          <GameBoardPiles
            drawPileCount={game?.drawPileCount ?? 0}
            topDiscardCard={topDiscardCard}
            canDrawDeck={turnState.canDrawDeck}
            canDrawDiscard={turnState.canDrawDiscard}
            canDiscard={turnState.canDiscard}
          />
          <GameBoardPlayers players={players} game={game} connectedPlayers={connectedPlayers} />
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
            canSubmitTablePlay,
          }}
          onResetTablePlay={onResetTablePlay}
          onSubmitTablePlay={onSubmitTablePlay}
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
  draftedCompositionsCount: number;
  canSubmitTablePlay: boolean;
  handleDragStart: (event: DragStartEvent) => void;
  handleDragOver: (event: DragOverEvent) => void;
  handleDragEnd: (event: DragEndEvent) => void;
  handleDragCancel: () => void;
  resetDraftCompositions: () => void;
  submitDraftCompositions: () => void;
};

function useGameBoardController({
  game,
  roomCode,
  playerId,
  turnState,
  topDiscardCard,
  onDragEnd,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
}: Pick<
  GameBoardViewProps,
  | "game"
  | "roomCode"
  | "playerId"
  | "turnState"
  | "topDiscardCard"
  | "onDragEnd"
  | "onDrawFromDeck"
  | "onDrawFromDiscard"
  | "onPlayTable"
>): GameBoardController {
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

  const activeEntry =
    activeDrag?.type === "hand"
      ? (handEntries.find((entry) => entry.key === activeDrag.handKey) ?? null)
      : null;
  const validHandKeys = useMemo(
    () => new Set(handEntries.map((entry) => entry.key)),
    [handEntries],
  );
  const draftCompositions = useMemo(
    () =>
      pruneDraftCompositions(
        draftCompositionState.scopeKey === currentDraftScopeKey
          ? draftCompositionState.compositions
          : [],
        validHandKeys,
      ),
    [
      currentDraftScopeKey,
      draftCompositionState.compositions,
      draftCompositionState.scopeKey,
      validHandKeys,
    ],
  );
  const stagedHandKeySet = useMemo(
    () => new Set(draftCompositions.flatMap((composition) => composition.handKeys)),
    [draftCompositions],
  );
  const availableHandEntries = useMemo(
    () => handEntries.filter((entry) => !stagedHandKeySet.has(entry.key)),
    [handEntries, stagedHandKeySet],
  );
  const sortableIds = useMemo(
    () => availableHandEntries.map((entry) => entry.key),
    [availableHandEntries],
  );
  const entryByKey = useMemo(
    () => new Map(handEntries.map((entry) => [entry.key, entry])),
    [handEntries],
  );
  const draftedCompositionsView = useMemo(
    () => buildDraftedCompositionViews(draftCompositions, entryByKey),
    [draftCompositions, entryByKey],
  );
  const tableCompositions = useMemo(
    () =>
      buildTableCompositionViews(
        game?.activeCompositions ?? [],
        draftedCompositionsView,
        entryByKey,
      ),
    [draftedCompositionsView, entryByKey, game?.activeCompositions],
  );
  const canCompose = turnState.canDiscard;
  const canSubmitTablePlay = canCompose && draftedCompositionsView.length > 0;

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

    onPlayTable(buildTablePlayRequest(draftedCompositionsView));
  }

  function handleDragStart(event: DragStartEvent) {
    const drawSource = event.active.data.current?.drawSource;

    if (drawSource === "deck") {
      setActiveDrag({
        type: "draw",
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
        card: topDiscardCard,
        baselineEntries: handEntries,
        baselineOrder: handEntryOrder(handEntries),
        revealedHandKey: null,
      });
      onDrawFromDiscard();
      return;
    }

    const handKey = typeof event.active.id === "string" ? event.active.id : null;
    setActiveDrag(handKey ? { type: "hand", handKey } : null);
  }

  function handleDragOver(event: DragOverEvent) {
    if (activeDrag?.type !== "draw" || activeDrawEntry === null) {
      return;
    }

    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

    if (overHandKey === null || overHandKey === "discard-pile") {
      updateHandOrder(activeDrag.baselineOrder);
      return;
    }

    updateHandOrder(handEntryOrder(moveHandEntry(handEntries, activeDrawEntry.key, overHandKey)));
  }

  function handleDragEnd(event: DragEndEvent) {
    const currentDrag = activeDrag;
    const draggedHandKey = typeof event.active.id === "string" ? event.active.id : null;
    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

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

    const overId = typeof event.over?.id === "string" ? event.over.id : null;
    const draggedEntry = entryByKey.get(draggedHandKey) ?? null;
    const droppedOnDraftCard =
      overId === null
        ? null
        : draftCompositions.find((composition) => composition.handKeys.includes(overId));
    const droppedOnHandCard =
      overId !== null && availableHandEntries.some((entry) => entry.key === overId);
    const droppedOnDraftContainer = overId ? compositionIdFromDropId(overId) : null;
    const droppedOnTableComposition = overId ? tableCompositionIndexFromDropId(overId) : null;

    if (event.over?.id === "discard-pile") {
      if (draftCompositions.some((composition) => composition.handKeys.includes(draggedHandKey))) {
        return;
      }

      onDragEnd(event);
      return;
    }

    if (!canCompose) {
      if (droppedOnHandCard && draggedHandKey !== overId) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, draggedHandKey, overId)));
      }
      return;
    }

    if (droppedOnTableComposition !== null && draggedEntry) {
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableComposition,
        );
        if (existing) {
          return insertHandKeyIntoDraft(current, draggedHandKey, existing.id);
        }

        const compositionId = `draft-${nextDraftIdRef.current}`;
        nextDraftIdRef.current += 1;

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          { id: compositionId, handKeys: [draggedHandKey], tableIndex: droppedOnTableComposition },
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
      updateDraftCompositions((current) =>
        insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftContainer),
      );
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
  onDragEnd,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
}: GameBoardViewProps) {
  const controller = useGameBoardController({
    game,
    roomCode,
    playerId,
    turnState,
    topDiscardCard,
    onDragEnd,
    onDrawFromDeck,
    onDrawFromDiscard,
    onPlayTable,
  });
  const activeDraw = controller.activeDrag?.type === "draw" ? controller.activeDrag : null;

  return (
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
        turnState={turnState}
        topDiscardCard={topDiscardCard}
        players={players}
        connectedPlayers={connectedPlayers}
        availableHandEntries={controller.availableHandEntries}
        sortableIds={controller.sortableIds}
        activeDrag={controller.activeDrag}
        draftedCompositionsCount={controller.draftedCompositionsCount}
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
  );
}

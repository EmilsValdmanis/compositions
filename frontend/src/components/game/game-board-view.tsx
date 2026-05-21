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
  buildTablePlayRequest,
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

function GameBoardLayout({
  game,
  tableCompositions,
  newCompositions,
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
  newCompositions: DraftedCompositionView[];
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
          newCompositions={newCompositions}
          canCompose={turnState.canDiscard}
          hasDraftedCompositions={draftedCompositionsCount > 0}
          canSubmitTablePlay={canSubmitTablePlay}
          onResetTablePlay={onResetTablePlay}
          onSubmitTablePlay={onSubmitTablePlay}
        />

        <div className="flex flex-col min-h-0 gap-4">
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
  const newCompositions = useMemo(
    () => draftedCompositionsView.filter((composition) => composition.tableIndex === null),
    [draftedCompositionsView],
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

      const cardIndex = event.active.data.current?.cardIndex;
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
  });
  const activeDraw = controller.activeDrag?.type === "draw" ? controller.activeDrag : null;

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
            <Button type="button" variant="outline" onClick={controller.closePendingDiscardDialog}>
              Keep editing
            </Button>
            <Button type="button" variant="ghost" onClick={controller.discardWithoutSubmitting}>
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

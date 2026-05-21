import { useEffect, useMemo, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  type CompositionDraftRequest,
  type GameSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { GameBoardBuilder } from "#/components/game/game-board-builder";
import { GameBoardHand } from "#/components/game/game-board-hand";
import { GameBoardPiles } from "#/components/game/game-board-piles";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { GameBoardTable } from "#/components/game/game-board-table";
import { GameCard } from "#/components/game/game-card";
import { playerName } from "#/components/game/game-view-utils";
import {
  FACE_DOWN_CARD,
  HAND_DROP_ID,
  NEW_COMPOSITION_DROP_ID,
  type ActiveDrag,
  type DraftComposition,
  buildHandEntries,
  compositionIdFromDropId,
  findNewHandEntry,
  insertHandKeyIntoDraft,
  moveHandEntry,
  removeHandKeyFromDrafts,
  reconcileHandEntries,
  sameStringArray,
} from "#/components/game/game-board-view-state";

export function GameBoardView({
  game,
  players,
  connectedPlayers,
  canDrawDeck,
  canDrawDiscard,
  canDiscard,
  isMyTurn,
  topDiscardCard,
  onDragEnd,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayCompositions,
}: {
  game: GameSnapshot | null;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
  isMyTurn: boolean;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  onDragEnd: (event: DragEndEvent) => void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
  onPlayCompositions: (compositions: CompositionDraftRequest[]) => void;
}) {
  const snapshotHandEntries = useMemo(() => buildHandEntries(game?.hand ?? []), [game?.hand]);
  const [handEntries, setHandEntries] = useState(snapshotHandEntries);
  const [activeDrag, setActiveDrag] = useState<ActiveDrag | null>(null);
  const [draftCompositions, setDraftCompositions] = useState<DraftComposition[]>([]);
  const nextDraftIdRef = useRef(0);

  useEffect(() => {
    if (activeDrag !== null) {
      return;
    }

    setHandEntries((current) => reconcileHandEntries(current, snapshotHandEntries));
  }, [activeDrag, snapshotHandEntries]);

  useEffect(() => {
    if (activeDrag?.type !== "draw" || activeDrag.revealedHandKey !== null) {
      return;
    }

    const drawnEntry = findNewHandEntry(handEntries, snapshotHandEntries);
    if (!drawnEntry) {
      return;
    }

    setHandEntries(snapshotHandEntries);
    setActiveDrag({ type: "draw", card: drawnEntry.card, revealedHandKey: drawnEntry.key });
  }, [activeDrag, handEntries, snapshotHandEntries]);

  const activeEntry =
    activeDrag?.type === "hand"
      ? (handEntries.find((entry) => entry.key === activeDrag.handKey) ?? null)
      : null;
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
    () =>
      draftCompositions.map((composition) => ({
        ...composition,
        entries: composition.handKeys
          .map((handKey) => entryByKey.get(handKey) ?? null)
          .filter((entry): entry is (typeof handEntries)[number] => entry !== null),
      })),
    [draftCompositions, entryByKey, handEntries],
  );
  const canCompose = canDiscard;
  const canSubmitCompositions = canCompose && draftedCompositionsView.length > 0;
  const turnPlayerName = playerName(players, game?.turn.playerId);

  useEffect(() => {
    const handKeySet = new Set(handEntries.map((entry) => entry.key));

    setDraftCompositions((current) => {
      let changed = false;
      const next = current
        .map((composition) => {
          const handKeys = composition.handKeys.filter((key) => handKeySet.has(key));
          if (!sameStringArray(composition.handKeys, handKeys)) {
            changed = true;
          }

          return {
            ...composition,
            handKeys,
          };
        })
        .filter((composition) => composition.handKeys.length > 0);

      if (!changed && next.length === current.length) {
        return current;
      }

      return next;
    });
  }, [handEntries]);

  useEffect(() => {
    if (game && isMyTurn) {
      return;
    }

    setDraftCompositions([]);
  }, [game, isMyTurn]);

  function resetDraftCompositions() {
    setDraftCompositions([]);
  }

  function updateDraftCompositionType(draftId: string, type: CompositionDraftRequest["type"]) {
    setDraftCompositions((current) =>
      current.map((composition) =>
        composition.id === draftId ? { ...composition, type } : composition,
      ),
    );
  }

  function submitDraftCompositions() {
    if (!canSubmitCompositions) {
      return;
    }

    onPlayCompositions(
      draftedCompositionsView.map((composition) => ({
        type: composition.type,
        cards: composition.entries.map((entry) => entry.card),
      })),
    );
  }

  function handleDragStart(event: DragStartEvent) {
    const drawSource = event.active.data.current?.drawSource;

    if (drawSource === "deck") {
      setActiveDrag({ type: "draw", card: null, revealedHandKey: null });
      onDrawFromDeck();
      return;
    }

    if (drawSource === "discard" && topDiscardCard) {
      setActiveDrag({ type: "draw", card: topDiscardCard, revealedHandKey: null });
      onDrawFromDiscard();
      return;
    }

    const handKey = typeof event.active.id === "string" ? event.active.id : null;
    setActiveDrag(handKey ? { type: "hand", handKey } : null);
  }

  function handleDragOver(event: DragOverEvent) {
    if (activeDrag?.type !== "draw" || activeDrag.revealedHandKey === null) {
      return;
    }

    const revealedHandKey = activeDrag.revealedHandKey;
    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

    if (overHandKey === null || overHandKey === "discard-pile") {
      setHandEntries(snapshotHandEntries);
      return;
    }

    setHandEntries((current) => moveHandEntry(current, revealedHandKey, overHandKey));
  }

  function handleDragEnd(event: DragEndEvent) {
    const currentDrag = activeDrag;
    const draggedHandKey = typeof event.active.id === "string" ? event.active.id : null;
    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

    setActiveDrag(null);

    if (currentDrag?.type === "draw") {
      if (currentDrag.revealedHandKey && overHandKey && overHandKey !== "discard-pile") {
        return;
      }

      setHandEntries(snapshotHandEntries);
      return;
    }

    if (currentDrag?.type !== "hand" || typeof draggedHandKey !== "string") {
      return;
    }

    const overId = typeof event.over?.id === "string" ? event.over.id : null;
    const droppedOnDraftCard =
      overId === null
        ? null
        : draftCompositions.find((composition) => composition.handKeys.includes(overId));
    const droppedOnHandCard =
      overId !== null && availableHandEntries.some((entry) => entry.key === overId);
    const droppedOnDraftContainer = overId ? compositionIdFromDropId(overId) : null;

    if (event.over?.id === "discard-pile") {
      if (draftCompositions.some((composition) => composition.handKeys.includes(draggedHandKey))) {
        return;
      }

      onDragEnd(event);
      return;
    }

    if (!canCompose) {
      if (droppedOnHandCard && draggedHandKey !== overId) {
        setHandEntries((current) => moveHandEntry(current, draggedHandKey, overId));
      }
      return;
    }

    if (overId === NEW_COMPOSITION_DROP_ID) {
      const compositionId = `draft-${nextDraftIdRef.current}`;
      nextDraftIdRef.current += 1;

      setDraftCompositions((current) => [
        ...removeHandKeyFromDrafts(current, draggedHandKey),
        { id: compositionId, type: "set", handKeys: [draggedHandKey] },
      ]);
      return;
    }

    if (droppedOnDraftCard) {
      setDraftCompositions((current) =>
        insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftCard.id, overId ?? undefined),
      );
      return;
    }

    if (droppedOnDraftContainer) {
      setDraftCompositions((current) =>
        insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftContainer),
      );
      return;
    }

    if (overId === HAND_DROP_ID) {
      setDraftCompositions((current) => removeHandKeyFromDrafts(current, draggedHandKey));
      return;
    }

    if (droppedOnHandCard) {
      setDraftCompositions((current) => removeHandKeyFromDrafts(current, draggedHandKey));

      if (draggedHandKey !== overId) {
        setHandEntries((current) => moveHandEntry(current, draggedHandKey, overId));
      }
    }
  }

  function handleDragCancel() {
    if (activeDrag?.type === "draw") {
      setHandEntries(snapshotHandEntries);
    }

    setActiveDrag(null);
  }

  return (
    <DndContext
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={handleDragCancel}
    >
      <div className="flex min-h-[calc(100vh-10.5rem)] flex-col gap-4">
        <div className="grid flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <GameBoardTable game={game} />

          <div className="grid content-start gap-4">
            <GameBoardPiles
              drawPileCount={game?.drawPileCount ?? 0}
              topDiscardCard={topDiscardCard}
              canDrawDeck={canDrawDeck}
              canDrawDiscard={canDrawDiscard}
              canDiscard={canDiscard}
            />
            <GameBoardPlayers players={players} game={game} connectedPlayers={connectedPlayers} />
          </div>
        </div>

        <div className="mt-auto grid gap-4">
          <GameBoardBuilder
            compositions={draftedCompositionsView}
            canSubmit={canSubmitCompositions}
            onReset={resetDraftCompositions}
            onSubmit={submitDraftCompositions}
            onTypeChange={updateDraftCompositionType}
          />
          <GameBoardHand
            hasGame={Boolean(game)}
            isMyTurn={isMyTurn}
            turnPlayerName={turnPlayerName}
            availableHandEntries={availableHandEntries}
            sortableIds={sortableIds}
            activeDrag={activeDrag}
            hasDraftedCompositions={draftedCompositionsView.length > 0}
          />
        </div>
      </div>
      <DragOverlay dropAnimation={null}>
        {activeDrag?.type === "draw" ? (
          <GameCard
            card={activeDrag.card ?? FACE_DOWN_CARD}
            faceDown={activeDrag.card === null}
            size="hand"
            className="rotate-3 shadow-xl ring-1 ring-foreground/10"
          />
        ) : activeEntry ? (
          <GameCard
            card={activeEntry.card}
            size="hand"
            className="rotate-3 shadow-xl ring-1 ring-foreground/10"
          />
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}

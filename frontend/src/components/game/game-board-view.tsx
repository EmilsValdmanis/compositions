import { useEffect, useMemo, useState } from "react";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, arrayMove, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  type CardSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { DiscardDropZone } from "#/components/game/discard-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { formatLabel, playerName } from "#/components/game/game-view-utils";
import { PlayerStrip } from "#/components/game/player-strip";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";

type HandEntry = {
  key: string;
  card: CardSnapshot;
  sourceIndex: number;
};

type ActiveDrag =
  | { type: "draw"; card: CardSnapshot | null; revealedHandKey: string | null }
  | { type: "hand"; handKey: string };

const FACE_DOWN_CARD: CardSnapshot = {};

function handCardKey(card: CardSnapshot) {
  if (card.isJoker) {
    return "joker";
  }

  return `${card.rank ?? "?"}-${card.suit ?? "?"}`;
}

function buildHandEntries(hand: CardSnapshot[]) {
  const counts = new Map<string, number>();

  return hand.map((card, sourceIndex) => {
    const baseKey = handCardKey(card);
    const occurrence = (counts.get(baseKey) ?? 0) + 1;
    counts.set(baseKey, occurrence);

    return {
      key: `${baseKey}-${occurrence}`,
      card,
      sourceIndex,
    } satisfies HandEntry;
  });
}

function reconcileHandEntries(current: HandEntry[], next: HandEntry[]) {
  const nextByKey = new Map(next.map((entry) => [entry.key, entry]));
  const ordered = current
    .map((entry) => nextByKey.get(entry.key) ?? null)
    .filter((entry): entry is HandEntry => entry !== null);
  const seenKeys = new Set(ordered.map((entry) => entry.key));

  return [...ordered, ...next.filter((entry) => !seenKeys.has(entry.key))];
}

function findNewHandEntry(current: HandEntry[], next: HandEntry[]) {
  const currentKeys = new Set(current.map((entry) => entry.key));
  return next.find((entry) => !currentKeys.has(entry.key)) ?? null;
}

function moveHandEntry(current: HandEntry[], handKey: string, overHandKey: string) {
  const oldIndex = current.findIndex((entry) => entry.key === handKey);
  const newIndex = current.findIndex((entry) => entry.key === overHandKey);

  if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
    return current;
  }

  return arrayMove(current, oldIndex, newIndex);
}

export function GameBoardView({
  game,
  phase,
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
}: {
  game: GameSnapshot | null;
  phase: string;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
  isMyTurn: boolean;
  topDiscardCard: CardSnapshot | null;
  onDragEnd: (event: DragEndEvent) => void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
}) {
  const snapshotHandEntries = useMemo(() => buildHandEntries(game?.hand ?? []), [game?.hand]);
  const [handEntries, setHandEntries] = useState<HandEntry[]>(snapshotHandEntries);
  const [activeDrag, setActiveDrag] = useState<ActiveDrag | null>(null);

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
  const sortableIds = useMemo(() => handEntries.map((entry) => entry.key), [handEntries]);

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

    if (currentDrag?.type !== "hand") {
      return;
    }

    if (event.over?.id === "discard-pile") {
      onDragEnd(event);
      return;
    }

    if (
      typeof draggedHandKey === "string" &&
      typeof overHandKey === "string" &&
      draggedHandKey !== overHandKey
    ) {
      setHandEntries((current) => moveHandEntry(current, draggedHandKey, overHandKey));
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
      <div className="flex gap-4">
        <div className="grid gap-4">
          <Card>
            <CardHeader>
              <CardTitle>Table</CardTitle>
              <CardDescription>
                Round {game?.round ?? 1} · Turn {game?.turn.number ?? 0} ·{" "}
                {playerName(players, game?.turn.playerId)}
              </CardDescription>
            </CardHeader>
            <CardContent className="grid min-h-64 gap-3">
              {game?.activeCompositions.length ? (
                game.activeCompositions.map((composition, index) => (
                  <CompositionRow key={index} composition={composition} index={index} />
                ))
              ) : (
                <div className="grid min-h-40 place-items-center rounded-3xl border border-dashed border-border/70 text-sm text-muted-foreground">
                  No compositions on the table.
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Hand</CardTitle>
              <CardDescription>
                {isMyTurn ? "Your turn" : `${playerName(players, game?.turn.playerId)} is up`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {game ? (
                <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
                  <div className="flex min-h-36 gap-3 overflow-auto pb-2">
                    {handEntries.map((entry) => (
                      <GameCard
                        key={entry.key}
                        card={entry.card}
                        size="hand"
                        draggable={{ id: entry.key, cardIndex: entry.sourceIndex }}
                        className={
                          activeDrag?.type === "draw" && entry.key === activeDrag.revealedHandKey
                            ? "invisible"
                            : undefined
                        }
                      />
                    ))}
                  </div>
                </SortableContext>
              ) : (
                <div className="rounded-3xl border border-dashed border-border/70 p-6 text-sm text-muted-foreground">
                  Waiting for the first game snapshot.
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="grid content-start gap-4">
          <Card>
            <CardHeader>
              <CardTitle>Turn</CardTitle>
              <CardDescription>{formatLabel(game?.phase ?? phase)}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3">
              <div className="grid gap-2 rounded-3xl border border-border/70 bg-muted/20 p-4 text-sm">
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Current player</span>
                  <span className="font-medium">{playerName(players, game?.turn.playerId)}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Drawn</span>
                  <span className="font-medium">{game?.turn.hasDrawn ? "Yes" : "No"}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Draw pile</span>
                  <span className="font-medium">{game?.drawPileCount ?? 0}</span>
                </div>
              </div>
              <p className="text-sm text-muted-foreground">
                Drag a pile card to draw. Discard draws must be used before you can throw a card
                away.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Piles</CardTitle>
              <CardDescription>Drop a hand card on discard after drawing.</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3">
              <div className="rounded-3xl border border-border/70 bg-muted/20 p-3">
                <p className="mb-2 text-xs uppercase text-muted-foreground">Draw</p>
                <div className="grid place-items-center gap-3 rounded-2xl border border-border bg-background p-3">
                  <GameCard
                    card={FACE_DOWN_CARD}
                    faceDown
                    dragSource={{
                      id: "draw-pile",
                      disabled: !canDrawDeck,
                      data: { drawSource: "deck" },
                    }}
                    className="shadow-md"
                  />
                  <span className="text-sm font-medium text-muted-foreground">
                    {game?.drawPileCount ?? 0} cards
                  </span>
                </div>
              </div>
              <DiscardDropZone disabled={!canDiscard}>
                <p className="mb-2 text-xs uppercase text-muted-foreground">Discard</p>
                <div className="grid min-h-28 place-items-center rounded-2xl border border-border bg-background p-3">
                  {topDiscardCard ? (
                    <GameCard
                      card={topDiscardCard}
                      dragSource={{
                        id: "discard-draw",
                        disabled: !canDrawDiscard,
                        data: { drawSource: "discard" },
                      }}
                      className={canDrawDiscard ? "shadow-md" : undefined}
                    />
                  ) : (
                    <span className="text-sm text-muted-foreground">Empty</span>
                  )}
                </div>
              </DiscardDropZone>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Players</CardTitle>
              <CardDescription>
                {connectedPlayers}/{players.length || 0} online
              </CardDescription>
            </CardHeader>
            <CardContent>
              <PlayerStrip players={players} game={game} />
            </CardContent>
          </Card>
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

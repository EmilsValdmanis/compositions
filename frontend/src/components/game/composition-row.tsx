import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import {
  type HandEntry,
  type PlannedJokerReclaim,
  tableCompositionDropId,
} from "#/components/game/game-board-view-state";
import {
  type CompositionSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import { cardName } from "#/components/game/game-card-utils";
import { formatLabel } from "#/components/game/game-view-helpers";
import { PlayerMarker } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";
import { cn } from "#/lib/utils";

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

export function CompositionRow({
  composition,
  index,
  stagedEntries = EMPTY_STAGED_ENTRIES,
  reclaims = EMPTY_RECLAIMS,
  players,
  activity,
}: {
  composition: CompositionSnapshot;
  index: number;
  stagedEntries?: HandEntry[];
  reclaims?: PlannedJokerReclaim[];
  players: PlayerSnapshot[];
  activity?: {
    kind?: string;
    playerId?: string;
    cardActivities?: Record<number, { kind: string; playerId: string }>;
  };
}) {
  const compositionCards = buildCompositionCardViews(composition.cards);
  const reclaimByJokerIndex = new Map(reclaims.map((reclaim) => [reclaim.jokerIndex, reclaim]));
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  const cardActivities = activity?.cardActivities ?? {};
  const isNewComposition = activity?.kind === "new_composition";

  return (
    <GameBoardDraftDropZone
      id={tableCompositionDropId(index)}
      className={cn(
        "flex min-w-0 flex-col rounded-3xl border border-border/70 bg-muted/20 p-3",
        isNewComposition ? "border-primary/40 bg-primary/5" : null,
      )}
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">#{index + 1}</Badge>
          <Badge variant="outline">{formatLabel(composition.type)}</Badge>
          {composition.complete ? <Badge>Complete</Badge> : null}
          {isNewComposition ? <Badge variant="outline">New</Badge> : null}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {activity?.playerId ? (
            <PlayerMarker players={players} playerId={activity.playerId} />
          ) : null}
          <span>{composition.points} pts</span>
        </div>
      </div>

      {reclaims.length > 0 ? (
        <div className="mb-3 rounded-2xl border border-primary/30 bg-primary/10 px-3 py-2 text-xs text-primary">
          {reclaims.length === 1
            ? `${cardName(reclaims[0].replacementEntry.card)} reclaims the joker. The joker returns to your hand after submit.`
            : `${reclaims.length} cards are reclaiming jokers. The reclaimed jokers return to your hand after submit.`}
        </div>
      ) : null}

      <div className="flex flex-wrap content-start justify-center gap-2">
        {compositionCards.map(({ card, index: cardIndex, key }) => {
          const reclaim = reclaimByJokerIndex.get(cardIndex);
          const cardActivity = cardActivities[cardIndex];

          return (
            <div key={key} className="flex flex-col items-center gap-1">
              <GameCard
                card={card}
                size="compact"
                decoration={{
                  highlight:
                    cardActivity?.kind === "new"
                      ? "new"
                      : cardActivity?.kind === "addition"
                        ? "addition"
                        : cardActivity?.kind === "joker_reclaim"
                          ? "joker_reclaim"
                          : reclaim
                            ? "joker_reclaim"
                            : undefined,
                  label:
                    cardActivity?.kind === "new"
                      ? "New"
                      : cardActivity?.kind === "addition"
                        ? "New"
                        : cardActivity?.kind === "joker_reclaim"
                          ? "Joker"
                          : undefined,
                  footer: cardActivity?.playerId ? (
                    <PlayerMarker players={players} playerId={cardActivity.playerId} />
                  ) : undefined,
                }}
              />
              {reclaim ? (
                <span className="text-center text-[0.6rem] leading-tight text-primary">
                  Replaced by {cardName(reclaim.replacementEntry.card)}
                </span>
              ) : null}
            </div>
          );
        })}
        <SortableContext
          items={stagedEntries.map((entry) => entry.key)}
          strategy={horizontalListSortingStrategy}
        >
          {stagedEntries.map((entry) => (
            <div key={entry.key} className="flex flex-col items-center gap-1">
              <GameCard
                card={entry.card}
                size="compact"
                draggable={{
                  id: entry.key,
                  cardIndex: entry.sourceIndex,
                  isVirtual: entry.isVirtual,
                }}
                className={cn(
                  reclaimedEntryKeys.has(entry.key)
                    ? "ring-2 ring-primary/60 ring-offset-1 ring-offset-card"
                    : null,
                )}
              />
              {reclaimedEntryKeys.has(entry.key) ? (
                <span className="text-center text-[0.6rem] leading-tight text-primary">
                  Reclaims joker
                </span>
              ) : null}
            </div>
          ))}
        </SortableContext>
      </div>
    </GameBoardDraftDropZone>
  );
}

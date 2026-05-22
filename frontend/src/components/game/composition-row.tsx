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
import { formatLabel } from "#/components/game/game-view-helpers";
import {
  ActivityLabel,
  NewActivityLabel,
  ReclaimActivityLabel,
} from "#/components/game/game-view-utils";
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
  stagedEntryPlayerId,
  stagedEntriesInteractive = true,
  activity,
}: {
  composition: CompositionSnapshot;
  index: number;
  stagedEntries?: HandEntry[];
  reclaims?: PlannedJokerReclaim[];
  players: PlayerSnapshot[];
  stagedEntryPlayerId?: string;
  stagedEntriesInteractive?: boolean;
  activity?: {
    kind?: string;
    playerId?: string;
    cardActivities?: Record<number, { kind: string; playerId: string }>;
  };
}) {
  const compositionCards = buildCompositionCardViews(composition.cards);
  const reclaimByJokerIndex = new Map(reclaims.map((reclaim) => [reclaim.jokerIndex, reclaim]));
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  const additionEntries = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));
  const cardActivities = activity?.cardActivities ?? {};
  const isNewComposition = activity?.kind === "new_composition";

  return (
    <GameBoardDraftDropZone
      id={tableCompositionDropId(index)}
      className={cn(
        "relative flex min-w-0 flex-col rounded-3xl border border-border/70 bg-muted/20 p-3",
        isNewComposition ? "border-primary/70 bg-primary/5" : null,
      )}
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">#{index + 1}</Badge>
          <Badge variant="outline">{formatLabel(composition.type)}</Badge>
          {composition.complete ? <Badge>Complete</Badge> : null}
          {isNewComposition ? (
            <NewActivityLabel players={players} playerId={activity?.playerId} />
          ) : null}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{composition.points} pts</span>
        </div>
      </div>

      <div className="flex flex-wrap content-start justify-center gap-2">
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

          return (
            <div key={key} className="flex flex-col items-center gap-1">
              <GameCard
                card={previewCard}
                size="compact"
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
                            <ReclaimActivityLabel players={players} playerId={previewPlayerId} />
                          ) : (
                            <ActivityLabel
                              players={players}
                              playerId={previewPlayerId}
                              label="Add"
                            />
                          ),
                      }
                    : undefined
                }
              />
            </div>
          );
        })}
        <SortableContext
          items={additionEntries.map((entry) => entry.key)}
          strategy={horizontalListSortingStrategy}
        >
          {additionEntries.map((entry) => (
            <div key={entry.key} className="flex flex-col items-center gap-1">
              <GameCard
                card={entry.card}
                size="compact"
                draggable={
                  stagedEntriesInteractive
                    ? {
                        id: entry.key,
                        cardIndex: entry.sourceIndex,
                        isVirtual: entry.isVirtual,
                      }
                    : undefined
                }
                decoration={{
                  highlight: "addition",
                  label: (
                    <ActivityLabel players={players} playerId={stagedEntryPlayerId} label="Add" />
                  ),
                }}
              />
            </div>
          ))}
        </SortableContext>
      </div>
    </GameBoardDraftDropZone>
  );
}

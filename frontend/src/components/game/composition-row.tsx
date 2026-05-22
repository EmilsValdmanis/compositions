import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { Tick01FreeIcons } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type HandEntry,
  type PlannedJokerReclaim,
  tableCompositionDropId,
  tableCompositionEdgeDropId,
} from "#/components/game/game-board-view-state";
import {
  type CompositionSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import { GameCard } from "#/components/game/game-card";
import {
  ActivityLabel,
  NewActivityLabel,
  ReclaimActivityLabel,
} from "#/components/game/game-view-utils";
import { cn } from "#/lib/utils";
import { Badge } from "../ui/badge";

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

function CompositionEdgeDraftZone({
  compositionIndex,
  edge,
  entries,
  interactive,
  players,
  playerId,
}: {
  compositionIndex: number;
  edge: "start" | "end";
  entries: HandEntry[];
  interactive: boolean;
  players: PlayerSnapshot[];
  playerId?: string;
}) {
  return (
    <GameBoardDraftDropZone
      id={tableCompositionEdgeDropId(compositionIndex, edge)}
      className={cn(
        "flex min-h-24 min-w-12 items-center justify-center rounded-2xl border border-dashed border-border/60 bg-background/20 px-2 py-2 transition",
        entries.length > 0 ? "border-primary/60 bg-primary/5" : "hover:border-primary/50",
      )}
    >
      <SortableContext items={entries.map((entry) => entry.key)} strategy={horizontalListSortingStrategy}>
        <div className="flex items-center gap-2">
          {entries.map((entry) => (
            <GameCard
              key={entry.key}
              card={entry.card}
              size="default"
              draggable={
                interactive
                  ? {
                      id: entry.key,
                      cardIndex: entry.sourceIndex,
                      isVirtual: entry.isVirtual,
                    }
                  : undefined
              }
              decoration={{
                highlight: "addition",
                label: <ActivityLabel players={players} playerId={playerId} label="Add" />,
              }}
            />
          ))}
        </div>
      </SortableContext>
    </GameBoardDraftDropZone>
  );
}

export function CompositionRow({
  composition,
  index,
  stagedEntries = EMPTY_STAGED_ENTRIES,
  reclaims = EMPTY_RECLAIMS,
  insertIndex,
  players,
  stagedEntryPlayerId,
  stagedEntriesInteractive = true,
  activity,
}: {
  composition: CompositionSnapshot;
  index: number;
  stagedEntries?: HandEntry[];
  reclaims?: PlannedJokerReclaim[];
  insertIndex?: number;
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
  const isHighlightedComposition = isNewComposition || composition.complete;
  const additionsAtStart = insertIndex === 0 ? additionEntries : EMPTY_STAGED_ENTRIES;
  const additionsAtEnd = insertIndex === 0 ? EMPTY_STAGED_ENTRIES : additionEntries;

  return (
    <GameBoardDraftDropZone
      id={tableCompositionDropId(index)}
      className={cn(
        "relative flex min-w-0 flex-col rounded-3xl border border-border/70 bg-muted/20 p-4",
        isHighlightedComposition ? "mt-5 border-primary/70 bg-primary/5" : null,
      )}
    >
      {composition.complete ? (
        <div className="absolute -top-1 -right-1 z-10 flex size-5 translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border border-primary/70 bg-primary/5 text-primary/70 shadow-sm">
          <HugeiconsIcon icon={Tick01FreeIcons} className="size-3" strokeWidth={2} />
        </div>
      ) : null}

      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">#{index + 1}</Badge>
          {isNewComposition ? (
            <NewActivityLabel players={players} playerId={activity?.playerId} />
          ) : null}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{composition.points} pts</span>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-center gap-3">
        <CompositionEdgeDraftZone
          compositionIndex={index}
          edge="start"
          entries={additionsAtStart}
          interactive={stagedEntriesInteractive}
          players={players}
          playerId={stagedEntryPlayerId}
        />

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
                size="default"
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
                            <ActivityLabel players={players} playerId={previewPlayerId} label="Add" />
                          ),
                      }
                    : undefined
                }
              />
            </div>
          );
        })}

        <CompositionEdgeDraftZone
          compositionIndex={index}
          edge="end"
          entries={additionsAtEnd}
          interactive={stagedEntriesInteractive}
          players={players}
          playerId={stagedEntryPlayerId}
        />
      </div>
    </GameBoardDraftDropZone>
  );
}

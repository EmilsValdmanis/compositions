import {
  buildHandEntries,
  type HandEntry,
  type PlannedJokerReclaim,
  type TableCompositionView,
  planTableJokerReclaims,
} from "#/components/game/game-board-view-state";
import { type DraftCompositionSnapshot } from "#/components/game-websocket-provider";

function alignedEntryIndexMap(
  entries: HandEntry[],
  placements: DraftCompositionSnapshot["cardPlacements"],
  field: "insertIndex" | "reclaimJokerIndex",
) {
  const normalized: Record<string, number> = {};

  for (const [index, entry] of entries.entries()) {
    const value = placements?.[index]?.[field];
    if (value !== undefined) {
      normalized[entry.key] = value;
    }
  }

  return normalized;
}

export function draftPreviewForComposition(
  tableComposition: TableCompositionView | undefined,
  draft: DraftCompositionSnapshot,
  cards: DraftCompositionSnapshot["cards"],
) {
  const stagedEntries = buildHandEntries(cards) as HandEntry[];

  if (!tableComposition) {
    return {
      stagedEntries,
      reclaims: [] as PlannedJokerReclaim[],
      insertIndex: draft.insertIndex,
      cardInsertIndices: undefined,
    };
  }

  const insertIndex = draft.insertIndex ?? tableComposition.snapshot.cards.length;
  const normalizedReclaimTargets = alignedEntryIndexMap(
    stagedEntries,
    draft.cardPlacements,
    "reclaimJokerIndex",
  );
  const normalizedCardInsertIndices = alignedEntryIndexMap(
    stagedEntries,
    draft.cardPlacements,
    "insertIndex",
  );

  const plannedReclaims = planTableJokerReclaims(
    tableComposition.snapshot,
    stagedEntries,
    insertIndex,
    normalizedReclaimTargets,
  );
  const reclaims = plannedReclaims.reclaims;
  const reclaimedEntryKeys = plannedReclaims.reclaimedEntryKeys;
  const additionEntries = stagedEntries.filter((entry) => !reclaimedEntryKeys.has(entry.key));
  const entriesByInsertIndex = new Map<number, HandEntry[]>();

  for (const entry of additionEntries) {
    const entryInsertIndex = normalizedCardInsertIndices[entry.key] ?? insertIndex;
    const entries = entriesByInsertIndex.get(entryInsertIndex) ?? [];
    entries.push(entry);
    entriesByInsertIndex.set(entryInsertIndex, entries);
  }

  const orderedAdditionEntries = [...(entriesByInsertIndex.get(0) ?? [])].reverse();
  for (const [entryInsertIndex, entries] of entriesByInsertIndex) {
    if (entryInsertIndex !== 0) {
      orderedAdditionEntries.push(...entries);
    }
  }

  return {
    stagedEntries: orderedAdditionEntries,
    reclaims,
    insertIndex,
    cardInsertIndices: normalizedCardInsertIndices,
  };
}

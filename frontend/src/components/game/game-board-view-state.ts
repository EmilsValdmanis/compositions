import { arrayMove } from "@dnd-kit/sortable";
import {
  type CardSnapshot,
  type CompositionSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";

export type HandEntry = {
  key: string;
  card: CardSnapshot;
  sourceIndex: number;
  isVirtual?: boolean;
};

export type ActiveDrag =
  | {
      type: "draw";
      source: "deck" | "discard";
      card: CardSnapshot | null;
      baselineEntries: HandEntry[];
      baselineOrder: string[];
      revealedHandKey: string | null;
    }
  | { type: "hand"; handKey: string };

export type DraftComposition = {
  id: string;
  handKeys: string[];
  tableIndex: number | null;
  insertIndex?: number;
  cardInsertIndices?: Record<string, number>;
};

export type DraftedCompositionView = DraftComposition & {
  entries: HandEntry[];
};

export type PlannedJokerReclaim = {
  jokerIndex: number;
  replacementEntry: HandEntry;
};

type JokerReclaimPlan = {
  additions: HandEntry[];
  reclaims: PlannedJokerReclaim[];
};

export type TableCompositionView = {
  tableIndex: number;
  key: string;
  snapshot: CompositionSnapshot;
  stagedEntries: HandEntry[];
  reclaims: PlannedJokerReclaim[];
  insertIndex: number;
  cardInsertIndices?: Record<string, number>;
};

export type TableCompositionEdge = "start" | "end";

export type VirtualReclaimedJoker = {
  key: string;
  jokerIndex: number;
  entry: HandEntry;
};

export const FACE_DOWN_CARD: CardSnapshot = {};
export const HAND_DROP_ID = "hand-drop-zone";
export const NEW_COMPOSITION_DROP_ID = "new-composition-drop-zone";

const DRAFT_COMPOSITION_DROP_ID_PREFIX = "draft-composition-";
const TABLE_COMPOSITION_DROP_ID_PREFIX = "table-composition-";
const TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX = "table-composition-edge-";

function handCardKey(card: CardSnapshot) {
  if (card.isJoker) {
    return "joker";
  }

  return `${card.rank ?? "?"}-${card.suit ?? "?"}`;
}

export function buildHandEntries(hand: CardSnapshot[]) {
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

export function reconcileHandEntries(current: HandEntry[], next: HandEntry[]) {
  const nextByKey = new Map(next.map((entry) => [entry.key, entry]));
  const ordered: HandEntry[] = [];
  const seenKeys = new Set<string>();

  for (const entry of current) {
    const nextEntry = nextByKey.get(entry.key);
    if (!nextEntry) {
      continue;
    }

    ordered.push(nextEntry);
    seenKeys.add(nextEntry.key);
  }

  for (const entry of next) {
    if (!seenKeys.has(entry.key)) {
      ordered.push(entry);
    }
  }

  return ordered;
}

export function applyHandEntryOrder(entries: HandEntry[], orderedKeys: string[]) {
  if (orderedKeys.length === 0) {
    return entries;
  }

  const entryByKey = new Map(entries.map((entry) => [entry.key, entry]));
  const ordered: HandEntry[] = [];
  const seenKeys = new Set<string>();

  for (const key of orderedKeys) {
    const entry = entryByKey.get(key);
    if (!entry) {
      continue;
    }

    ordered.push(entry);
    seenKeys.add(entry.key);
  }

  for (const entry of entries) {
    if (!seenKeys.has(entry.key)) {
      ordered.push(entry);
    }
  }

  return ordered;
}

export function findNewHandEntry(current: HandEntry[], next: HandEntry[]) {
  const currentKeys = new Set(current.map((entry) => entry.key));
  return next.find((entry) => !currentKeys.has(entry.key)) ?? null;
}

export function moveHandEntry(current: HandEntry[], handKey: string, overHandKey: string) {
  const oldIndex = current.findIndex((entry) => entry.key === handKey);
  const newIndex = current.findIndex((entry) => entry.key === overHandKey);

  if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
    return current;
  }

  return arrayMove(current, oldIndex, newIndex);
}

export function sameStringArray(a: string[], b: string[]) {
  return a.length === b.length && a.every((value, index) => value === b[index]);
}

export function draftCompositionDropId(compositionId: string) {
  return `${DRAFT_COMPOSITION_DROP_ID_PREFIX}${compositionId}`;
}

export function compositionIdFromDropId(dropId: string) {
  if (!dropId.startsWith(DRAFT_COMPOSITION_DROP_ID_PREFIX)) {
    return null;
  }

  return dropId.slice(DRAFT_COMPOSITION_DROP_ID_PREFIX.length);
}

export function tableCompositionDropId(compositionIndex: number) {
  return `${TABLE_COMPOSITION_DROP_ID_PREFIX}${compositionIndex}`;
}

export function tableCompositionEdgeDropId(compositionIndex: number, edge: TableCompositionEdge) {
  return `${TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX}${compositionIndex}-${edge}`;
}

export function tableCompositionIndexFromDropId(dropId: string) {
  if (!dropId.startsWith(TABLE_COMPOSITION_DROP_ID_PREFIX)) {
    return null;
  }

  const compositionIndex = Number(dropId.slice(TABLE_COMPOSITION_DROP_ID_PREFIX.length));
  return Number.isInteger(compositionIndex) && compositionIndex >= 0 ? compositionIndex : null;
}

export function tableCompositionEdgeTargetFromDropId(dropId: string) {
  if (!dropId.startsWith(TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX)) {
    return null;
  }

  const target = dropId.slice(TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX.length);
  const separatorIndex = target.indexOf("-");
  if (separatorIndex < 0) {
    return null;
  }

  const compositionIndex = Number(target.slice(0, separatorIndex));
  const edge = target.slice(separatorIndex + 1);

  if (!Number.isInteger(compositionIndex) || compositionIndex < 0) {
    return null;
  }

  if (edge !== "start" && edge !== "end") {
    return null;
  }

  return {
    compositionIndex,
    edge,
  } as const;
}

export function removeHandKeyFromDrafts(compositions: DraftComposition[], handKey: string) {
  const next: DraftComposition[] = [];

  for (const composition of compositions) {
    const handKeys: string[] = [];

    for (const key of composition.handKeys) {
      if (key !== handKey) {
        handKeys.push(key);
      }
    }

    if (handKeys.length > 0) {
      next.push({
        ...composition,
        handKeys,
      });
    }
  }

  return next;
}

export function handEntryOrder(entries: HandEntry[]) {
  const order: string[] = [];

  for (const entry of entries) {
    order.push(entry.key);
  }

  return order;
}

export function mapHandKeysToEntries(handKeys: string[], entryByKey: Map<string, HandEntry>) {
  const entries: HandEntry[] = [];

  for (const handKey of handKeys) {
    const entry = entryByKey.get(handKey);
    if (entry) {
      entries.push(entry);
    }
  }

  return entries;
}

export function pruneDraftCompositions(
  compositions: DraftComposition[],
  validHandKeys: Set<string>,
) {
  const next: DraftComposition[] = [];

  for (const composition of compositions) {
    const handKeys: string[] = [];

    for (const handKey of composition.handKeys) {
      if (validHandKeys.has(handKey)) {
        handKeys.push(handKey);
      }
    }

    if (handKeys.length > 0) {
      next.push({
        ...composition,
        handKeys,
      });
    }
  }

  return next;
}

export function insertHandKeyIntoDraft(
  compositions: DraftComposition[],
  handKey: string,
  targetCompositionId: string,
  overHandKey?: string,
) {
  const sourceComposition = compositions.find((composition) =>
    composition.handKeys.includes(handKey),
  );
  const targetComposition = compositions.find(
    (composition) => composition.id === targetCompositionId,
  );

  if (
    sourceComposition &&
    targetComposition &&
    sourceComposition.id === targetComposition.id &&
    overHandKey
  ) {
    const oldIndex = sourceComposition.handKeys.indexOf(handKey);
    const newIndex = sourceComposition.handKeys.indexOf(overHandKey);

    if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
      return compositions;
    }

    return compositions.map((composition) =>
      composition.id === sourceComposition.id
        ? {
            ...composition,
            handKeys: arrayMove(composition.handKeys, oldIndex, newIndex),
          }
        : composition,
    );
  }

  const next = removeHandKeyFromDrafts(compositions, handKey);
  const targetIndex = next.findIndex((composition) => composition.id === targetCompositionId);

  if (targetIndex < 0) {
    return next;
  }

  const target = next[targetIndex];
  const handKeys = target.handKeys.filter((key) => key !== handKey);
  const insertIndex = overHandKey ? handKeys.indexOf(overHandKey) : -1;

  next[targetIndex] = {
    ...target,
    handKeys:
      insertIndex >= 0
        ? [...handKeys.slice(0, insertIndex), handKey, ...handKeys.slice(insertIndex)]
        : [...handKeys, handKey],
  };

  return next;
}

export function moveDraftCompositionInsertIndex(
  compositions: DraftComposition[],
  compositionId: string,
  insertIndex: number,
) {
  return compositions.map((composition) =>
    composition.id === compositionId
      ? {
          ...composition,
          insertIndex,
        }
      : composition,
  );
}

export function setDraftCardInsertIndex(
  draftCompositions: DraftComposition[],
  compositionId: string,
  handKey: string,
  insertIndex: number,
): DraftComposition[] {
  return draftCompositions.map((composition) => {
    if (composition.id !== compositionId) {
      return composition;
    }
    return {
      ...composition,
      cardInsertIndices: {
        ...composition.cardInsertIndices,
        [handKey]: insertIndex,
      },
    };
  });
}

function cardsEqual(a: CardSnapshot, b: CardSnapshot) {
  return (
    Boolean(a.isJoker) === Boolean(b.isJoker) &&
    (a.rank ?? null) === (b.rank ?? null) &&
    (a.suit ?? null) === (b.suit ?? null)
  );
}

function jokerReclaimOptions(composition: CompositionSnapshot) {
  const options: Array<{ jokerIndex: number; replacementCard: CardSnapshot }> = [];

  for (const [indexKey, cards] of Object.entries(composition.jokerRepresentations ?? {})) {
    const jokerIndex = Number(indexKey);
    const replacementCard = cards[0];

    if (
      Number.isInteger(jokerIndex) &&
      jokerIndex >= 0 &&
      composition.cards[jokerIndex]?.isJoker &&
      cards.length === 1 &&
      replacementCard
    ) {
      options.push({ jokerIndex, replacementCard });
    }
  }

  return options;
}

function isAmbiguousRunInsertion(composition: CompositionSnapshot, insertIndex: number) {
  if (composition.type !== "run") {
    return false;
  }

  return insertIndex > 0 && insertIndex < composition.cards.length;
}

export function tableCompositionInsertIndexForEdge(
  composition: CompositionSnapshot,
  edge: TableCompositionEdge,
) {
  return edge === "start" ? 0 : composition.cards.length;
}

function inferSetPlannedJokerReclaims(
  composition: CompositionSnapshot,
  stagedEntries: HandEntry[],
): JokerReclaimPlan {
  const nonJokerCards = composition.cards.filter((card) => !card.isJoker);
  const setRank = nonJokerCards[0]?.rank ?? null;

  if (setRank === null) {
    return {
      additions: stagedEntries,
      reclaims: [],
    };
  }

  const presentSuits = new Set(
    nonJokerCards
      .map((card) => card.suit)
      .filter((suit): suit is number => typeof suit === "number"),
  );
  const jokerIndices = composition.cards.flatMap((card, index) => (card.isJoker ? [index] : []));
  const remainingEntries: HandEntry[] = [];
  const additions: HandEntry[] = [];
  let additionsRemaining = Math.max(0, 4 - composition.cards.length);

  for (const entry of stagedEntries) {
    const suit = entry.card.suit;
    if (
      entry.card.isJoker ||
      entry.card.rank !== setRank ||
      typeof suit !== "number" ||
      presentSuits.has(suit)
    ) {
      remainingEntries.push(entry);
      continue;
    }

    if (additionsRemaining > 0) {
      additions.push(entry);
      presentSuits.add(suit);
      additionsRemaining -= 1;
      continue;
    }

    remainingEntries.push(entry);
  }

  const availableOptions = jokerReclaimOptions(composition);
  if (availableOptions.length === 0 && jokerIndices.length === 1) {
    const missingSuits = [0, 1, 2, 3].filter((suit) => !presentSuits.has(suit));
    if (missingSuits.length === 1) {
      availableOptions.push({
        jokerIndex: jokerIndices[0],
        replacementCard: { rank: setRank, suit: missingSuits[0] },
      });
    }
  }

  const reclaims: PlannedJokerReclaim[] = [];
  const optionIndexByCardKey = new Map(
    availableOptions.map((option) => [
      `${option.replacementCard.rank ?? "?"}-${option.replacementCard.suit ?? "?"}-${Boolean(option.replacementCard.isJoker)}`,
      option,
    ]),
  );

  for (const entry of remainingEntries) {
    const matchedOption = optionIndexByCardKey.get(
      `${entry.card.rank ?? "?"}-${entry.card.suit ?? "?"}-${Boolean(entry.card.isJoker)}`,
    );

    if (!matchedOption || !cardsEqual(matchedOption.replacementCard, entry.card)) {
      additions.push(entry);
      continue;
    }

    optionIndexByCardKey.delete(
      `${matchedOption.replacementCard.rank ?? "?"}-${matchedOption.replacementCard.suit ?? "?"}-${Boolean(matchedOption.replacementCard.isJoker)}`,
    );

    reclaims.push({
      jokerIndex: matchedOption.jokerIndex,
      replacementEntry: entry,
    });
  }

  return {
    additions,
    reclaims,
  };
}

function inferPlannedJokerReclaimsForInsertIndex(
  composition: CompositionSnapshot,
  stagedEntries: HandEntry[],
  insertIndex: number,
): JokerReclaimPlan {
  if (composition.type === "set") {
    return inferSetPlannedJokerReclaims(composition, stagedEntries);
  }

  const availableOptions = jokerReclaimOptions(composition);

  if (availableOptions.length === 0 || isAmbiguousRunInsertion(composition, insertIndex)) {
    return {
      additions: stagedEntries,
      reclaims: [],
    };
  }

  const reclaims: PlannedJokerReclaim[] = [];
  const additions: HandEntry[] = [];
  const optionIndexByCardKey = new Map(
    availableOptions.map((option, optionIndex) => [
      `${option.replacementCard.rank ?? "?"}-${option.replacementCard.suit ?? "?"}-${Boolean(option.replacementCard.isJoker)}`,
      optionIndex,
    ]),
  );

  for (const entry of stagedEntries) {
    const cardKey = `${entry.card.rank ?? "?"}-${entry.card.suit ?? "?"}-${Boolean(entry.card.isJoker)}`;
    const matchedOptionIndex = optionIndexByCardKey.get(cardKey) ?? -1;

    if (matchedOptionIndex < 0) {
      additions.push(entry);
      continue;
    }

    const [matchedOption] = availableOptions.splice(matchedOptionIndex, 1);
    if (!matchedOption) {
      additions.push(entry);
      continue;
    }

    optionIndexByCardKey.delete(cardKey);

    reclaims.push({
      jokerIndex: matchedOption.jokerIndex,
      replacementEntry: entry,
    });
  }

  return {
    additions,
    reclaims,
  };
}

export function inferPlannedJokerReclaims(
  composition: CompositionSnapshot,
  stagedEntries: HandEntry[],
  insertIndex = composition.cards.length,
) {
  return inferPlannedJokerReclaimsForInsertIndex(composition, stagedEntries, insertIndex);
}

function buildOrderedAdditionEntries(
  composition: CompositionSnapshot,
  stagedEntries: HandEntry[],
  insertIndex: number,
) {
  const { additions, reclaims } = inferPlannedJokerReclaimsForInsertIndex(
    composition,
    stagedEntries,
    insertIndex,
  );

  if (reclaims.length === 0) {
    return {
      additions,
      reclaims,
      orderedEntries: additions,
    };
  }

  return {
    additions,
    reclaims,
    orderedEntries: additions,
  };
}

export function buildTablePlayRequest(
  activeCompositions: CompositionSnapshot[],
  draftedCompositionsView: DraftedCompositionView[],
): TablePlayRequest {
  const compositions: TablePlayRequest["compositions"] = [];
  const additions: TablePlayRequest["additions"] = [];
  const reclaims: TablePlayRequest["reclaims"] = [];

  for (const composition of draftedCompositionsView) {
    if (composition.tableIndex === null) {
      compositions.push({
        cards: composition.entries.map((entry) => entry.card),
      });
      continue;
    }

    const activeComposition = activeCompositions[composition.tableIndex];
    const cardInsertIndices = composition.cardInsertIndices;
    const defaultInsertIndex = composition.insertIndex ?? activeComposition?.cards.length ?? 0;

    const entriesByInsertIndex = new Map<number, HandEntry[]>();
    for (const entry of composition.entries) {
      const idx = cardInsertIndices?.[entry.key] ?? defaultInsertIndex;
      const group = entriesByInsertIndex.get(idx);
      if (group) {
        group.push(entry);
      } else {
        entriesByInsertIndex.set(idx, [entry]);
      }
    }

    for (const [insertIndex, entries] of entriesByInsertIndex) {
      const { reclaims: stagedReclaims, orderedEntries } = activeComposition
        ? buildOrderedAdditionEntries(activeComposition, entries, insertIndex)
        : { reclaims: [], orderedEntries: entries };

      if (orderedEntries.length > 0) {
        additions.push({
          compositionIndex: composition.tableIndex,
          insertIndex,
          cards: orderedEntries.map((entry) => entry.card),
        });
      }

      for (const reclaim of stagedReclaims) {
        reclaims.push({
          compositionIndex: composition.tableIndex,
          jokerIndex: reclaim.jokerIndex,
          replacementCard: reclaim.replacementEntry.card,
        });
      }
    }
  }

  return {
    compositions,
    additions,
    reclaims,
  };
}

export function buildTableCompositionViews(
  activeCompositions: CompositionSnapshot[],
  draftCompositions: DraftComposition[],
  entryByKey: Map<string, HandEntry>,
) {
  const views: TableCompositionView[] = activeCompositions.map((composition, index) => ({
    tableIndex: index,
    key: `table-${index}`,
    snapshot: composition,
    stagedEntries: [],
    reclaims: [],
    insertIndex: composition.cards.length,
  }));

  for (const composition of draftCompositions) {
    if (composition.tableIndex === null) {
      continue;
    }

    const existing = views[composition.tableIndex];
    if (existing) {
      existing.stagedEntries = mapHandKeysToEntries(composition.handKeys, entryByKey);
      const insertIndex = composition.insertIndex ?? existing.snapshot.cards.length;
      existing.insertIndex = insertIndex;
      existing.cardInsertIndices = composition.cardInsertIndices;
      existing.reclaims = inferPlannedJokerReclaims(
        existing.snapshot,
        existing.stagedEntries,
        insertIndex,
      ).reclaims;
    }
  }

  return views;
}

export function buildVirtualReclaimedJokers(tableCompositions: TableCompositionView[]) {
  const virtualJokers: VirtualReclaimedJoker[] = [];

  for (const composition of tableCompositions) {
    for (const reclaim of composition.reclaims) {
      virtualJokers.push({
        key: `reclaimed-joker-${composition.tableIndex}-${reclaim.jokerIndex}`,
        jokerIndex: reclaim.jokerIndex,
        entry: {
          key: `reclaimed-joker-${composition.tableIndex}-${reclaim.jokerIndex}`,
          card: { isJoker: true },
          sourceIndex: -1,
          isVirtual: true,
        },
      });
    }
  }

  return virtualJokers;
}

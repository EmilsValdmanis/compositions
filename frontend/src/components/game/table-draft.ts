import { buildHandEntries, cardsEqual, type HandEntry } from "#/components/game/card";
import {
  type CardSnapshot,
  type CompositionSnapshot,
  type DraftCompositionSnapshot,
} from "#/lib/game-protocol";

type JokerReclaimOption = { jokerIndex: number; replacementCard: CardSnapshot };

type PlannedJokerReclaim = { jokerIndex: number; replacementEntry: HandEntry };

export type TableDraftPlan = {
  additions: Array<{ insertIndex: number; entries: HandEntry[] }>;
  reclaims: PlannedJokerReclaim[];
};

function jokerReclaimOptions(composition: CompositionSnapshot) {
  const options: JokerReclaimOption[] = [];
  for (const [index, cards] of Object.entries(composition.jokerRepresentations ?? {})) {
    const jokerIndex = Number(index);
    if (
      Number.isInteger(jokerIndex) &&
      composition.cards[jokerIndex]?.isJoker &&
      cards.length === 1 &&
      cards[0]
    ) {
      options.push({ jokerIndex, replacementCard: cards[0] });
    }
  }
  return options;
}

function matchingReclaim(options: JokerReclaimOption[], jokerIndex: number, card: CardSnapshot) {
  return options.some(
    (option) => option.jokerIndex === jokerIndex && cardsEqual(option.replacementCard, card),
  );
}

export function canReclaimJokerWithCard(
  composition: CompositionSnapshot,
  jokerIndex: number,
  card: CardSnapshot,
) {
  return matchingReclaim(jokerReclaimOptions(composition), jokerIndex, card);
}

function inferSetReclaims(
  composition: CompositionSnapshot,
  entries: HandEntry[],
  options: JokerReclaimOption[],
) {
  const naturals = composition.cards.filter((card) => !card.isJoker);
  const rank = naturals[0]?.rank;
  if (rank == null) return [];

  const presentSuits = new Set(naturals.map((card) => card.suit));
  let remainingSlots = Math.max(0, 4 - composition.cards.length);
  const reclaimCandidates: HandEntry[] = [];

  for (const entry of entries) {
    const suit = entry.card.suit;
    if (
      remainingSlots > 0 &&
      !entry.card.isJoker &&
      entry.card.rank === rank &&
      typeof suit === "number" &&
      !presentSuits.has(suit)
    ) {
      remainingSlots -= 1;
      presentSuits.add(suit);
    } else {
      reclaimCandidates.push(entry);
    }
  }

  const available = options.slice();
  const jokerIndices = composition.cards.flatMap((card, index) => (card.isJoker ? [index] : []));
  if (available.length === 0 && jokerIndices.length === 1) {
    const missingSuits = [0, 1, 2, 3].filter((suit) => !presentSuits.has(suit));
    if (missingSuits.length === 1) {
      available.push({
        jokerIndex: jokerIndices[0],
        replacementCard: { rank, suit: missingSuits[0] },
      });
    }
  }

  const reclaims: PlannedJokerReclaim[] = [];
  for (const entry of reclaimCandidates) {
    const index = available.findIndex((option) => cardsEqual(option.replacementCard, entry.card));
    if (index !== -1) {
      const [option] = available.splice(index, 1);
      reclaims.push({ jokerIndex: option.jokerIndex, replacementEntry: entry });
    }
  }
  return reclaims;
}

function planReclaims(
  composition: CompositionSnapshot,
  entries: HandEntry[],
  targets: Record<string, number> = {},
) {
  const options = jokerReclaimOptions(composition);
  const reclaims: PlannedJokerReclaim[] = [];
  const usedJokerIndices = new Set<number>();
  for (const entry of entries) {
    const jokerIndex = targets[entry.key];
    if (
      typeof jokerIndex === "number" &&
      !usedJokerIndices.has(jokerIndex) &&
      matchingReclaim(options, jokerIndex, entry.card)
    ) {
      usedJokerIndices.add(jokerIndex);
      reclaims.push({ jokerIndex, replacementEntry: entry });
    }
  }

  if (reclaims.length > 0 || composition.type !== "set") {
    return reclaims;
  }

  // Fill available set slots before trying an explicitly targeted, ambiguous joker.
  return inferSetReclaims(
    composition,
    [
      ...entries.filter((entry) => targets[entry.key] === undefined),
      ...entries.filter((entry) => targets[entry.key] !== undefined),
    ],
    options,
  );
}

export function planTableDraft(
  composition: CompositionSnapshot | undefined,
  draft: {
    entries: HandEntry[];
    insertIndex?: number;
    cardInsertIndices?: Record<string, number>;
    reclaimTargets?: Record<string, number>;
  },
): TableDraftPlan {
  const reclaims = composition
    ? planReclaims(composition, draft.entries, draft.reclaimTargets)
    : [];
  const reclaimedEntryKeys = new Set(reclaims.map((reclaim) => reclaim.replacementEntry.key));
  const defaultInsertIndex = draft.insertIndex ?? composition?.cards.length ?? 0;
  const groups = new Map<number, HandEntry[]>();

  for (const entry of draft.entries) {
    if (reclaimedEntryKeys.has(entry.key)) continue;

    const insertIndex = draft.cardInsertIndices?.[entry.key] ?? defaultInsertIndex;
    const entries = groups.get(insertIndex) ?? [];
    entries.push(entry);
    groups.set(insertIndex, entries);
  }

  // The server applies groups in order, so work back from the last index to keep earlier indices valid.
  const additions = Array.from(groups, ([insertIndex, entries]) => ({
    insertIndex,
    entries: insertIndex === 0 ? entries.reverse() : entries,
  })).sort((left, right) => right.insertIndex - left.insertIndex);
  return { additions, reclaims };
}

export function cardsWithTableDraft(composition: CompositionSnapshot, draft: TableDraftPlan) {
  const replacements = new Map(
    draft.reclaims.map(({ jokerIndex, replacementEntry }) => [jokerIndex, replacementEntry.card]),
  );
  const cards = composition.cards.map((card, index) => replacements.get(index) ?? card);
  for (const { insertIndex, entries } of draft.additions) {
    cards.splice(insertIndex, 0, ...entries.map((entry) => entry.card));
  }
  return cards;
}

export function draftPreviewForComposition(
  composition: CompositionSnapshot,
  draft: DraftCompositionSnapshot,
) {
  const entries = buildHandEntries(draft.cards);
  const cardInsertIndices: Record<string, number> = {};
  const reclaimTargets: Record<string, number> = {};
  for (const [index, entry] of entries.entries()) {
    const placement = draft.cardPlacements?.[index];
    if (placement?.insertIndex !== undefined) cardInsertIndices[entry.key] = placement.insertIndex;
    if (placement?.reclaimJokerIndex !== undefined)
      reclaimTargets[entry.key] = placement.reclaimJokerIndex;
  }
  return planTableDraft(composition, {
    entries,
    insertIndex: draft.insertIndex,
    cardInsertIndices,
    reclaimTargets,
  });
}

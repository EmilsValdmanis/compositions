import { type TableCompositionEdge } from "#/components/game/board-drop-targets";
import { type HandEntry } from "#/components/game/card";
import { arrayMove } from "@dnd-kit/sortable";
import {
  type CardSnapshot,
  type CompositionSnapshot,
  type DraftCompositionSnapshot,
  type TablePlayRequest,
} from "#/lib/game-protocol";
import {
  cardsWithTableDraft,
  planTableDraft,
  type TableDraftPlan,
} from "#/components/game/table-draft";
import { isValidDraftComposition } from "#/components/game/game-card-utils";

export type ActiveDrag =
  | {
      type: "draw";
      source: "deck" | "discard";
      card: CardSnapshot | null;
      baselineEntries: HandEntry[];
      baselineOrder: string[];
      revealedHandKey: string | null;
    }
  | {
      type: "hand";
      handKey: string;
      baselineHandOrder: string[];
      baselineDraftCompositions: DraftComposition[];
      hasReorderedHand: boolean;
    };

export type DraftComposition = {
  id: string;
  handKeys: string[];
  tableIndex: number | null;
  insertIndex?: number;
  cardInsertIndices?: Record<string, number>;
  reclaimTargets?: Record<string, number>;
};

export type DraftedCompositionView = DraftComposition & {
  entries: HandEntry[];
};

export type TableCompositionView = {
  tableIndex: number;
  key: string;
  snapshot: CompositionSnapshot;
  draft?: TableDraftPlan;
};

export type VirtualReclaimedJoker = {
  key: string;
  jokerIndex: number;
  entry: HandEntry;
};

export type OpeningTablePlayValidation = {
  canSubmit: boolean;
  reason?: "missing-own-composition";
};

export function buildDraftCompositionSnapshot(
  composition: DraftedCompositionView,
): DraftCompositionSnapshot {
  const cardPlacements = composition.entries.map((entry) => ({
    insertIndex: composition.cardInsertIndices?.[entry.key],
    reclaimJokerIndex: composition.reclaimTargets?.[entry.key],
  }));
  const hasCardPlacements = cardPlacements.some(
    (placement) => placement.insertIndex !== undefined || placement.reclaimJokerIndex !== undefined,
  );

  return {
    id: composition.id,
    tableIndex: composition.tableIndex ?? undefined,
    insertIndex: composition.insertIndex,
    cardPlacements: hasCardPlacements ? cardPlacements : undefined,
    cards: composition.entries.map((entry) => entry.card),
  };
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

export function handIndexAfterSubmittedDrafts(
  rawHandEntries: HandEntry[],
  handKey: string,
  draftedCompositions: DraftedCompositionView[],
) {
  const draggedEntry = rawHandEntries.find((entry) => entry.key === handKey);

  if (!draggedEntry || draggedEntry.isVirtual) {
    return null;
  }

  const playedSourceIndices = new Set<number>();
  for (const composition of draftedCompositions) {
    for (const entry of composition.entries) {
      if (!entry.isVirtual) {
        playedSourceIndices.add(entry.sourceIndex);
      }
    }
  }

  if (playedSourceIndices.has(draggedEntry.sourceIndex)) {
    return null;
  }

  let nextIndex = 0;
  for (const entry of rawHandEntries) {
    if (entry.isVirtual) {
      continue;
    }
    if (entry.sourceIndex === draggedEntry.sourceIndex) {
      return nextIndex;
    }
    if (!playedSourceIndices.has(entry.sourceIndex)) {
      nextIndex += 1;
    }
  }

  return null;
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

export function removeHandKeyFromDrafts(compositions: DraftComposition[], handKey: string) {
  if (!compositions.some((composition) => composition.handKeys.includes(handKey))) {
    return compositions;
  }

  const next: DraftComposition[] = [];

  for (const composition of compositions) {
    const handKeys: string[] = [];

    for (const key of composition.handKeys) {
      if (key !== handKey) {
        handKeys.push(key);
      }
    }

    if (handKeys.length > 0) {
      const nextCardInsertIndices = Object.fromEntries(
        Object.entries(composition.cardInsertIndices ?? {}).filter(([key]) => key !== handKey),
      );
      const nextReclaimTargets = Object.fromEntries(
        Object.entries(composition.reclaimTargets ?? {}).filter(([key]) => key !== handKey),
      );

      next.push({
        ...composition,
        handKeys,
        cardInsertIndices:
          Object.keys(nextCardInsertIndices).length > 0 ? nextCardInsertIndices : undefined,
        reclaimTargets: Object.keys(nextReclaimTargets).length > 0 ? nextReclaimTargets : undefined,
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

function mapHandKeysToEntries(handKeys: string[], entryByKey: Map<string, HandEntry>) {
  const entries: HandEntry[] = [];

  for (const handKey of handKeys) {
    const entry = entryByKey.get(handKey);
    if (entry) {
      entries.push(entry);
    }
  }

  return entries;
}

function pruneDraftCompositions(compositions: DraftComposition[], validHandKeys: Set<string>) {
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

  const next = [...removeHandKeyFromDrafts(compositions, handKey)];
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
      reclaimTargets: Object.fromEntries(
        Object.entries(composition.reclaimTargets ?? {}).filter(([key]) => key !== handKey),
      ),
    };
  });
}

export function setDraftCardReclaimTarget(
  draftCompositions: DraftComposition[],
  compositionId: string,
  handKey: string,
  jokerIndex: number,
): DraftComposition[] {
  return draftCompositions.map((composition) => {
    if (composition.id !== compositionId) {
      return composition;
    }

    const nextCardInsertIndices = Object.fromEntries(
      Object.entries(composition.cardInsertIndices ?? {}).filter(([key]) => key !== handKey),
    );

    return {
      ...composition,
      cardInsertIndices:
        Object.keys(nextCardInsertIndices).length > 0 ? nextCardInsertIndices : undefined,
      reclaimTargets: {
        ...composition.reclaimTargets,
        [handKey]: jokerIndex,
      },
    };
  });
}

export function tableCompositionInsertIndexForEdge(
  composition: CompositionSnapshot,
  edge: TableCompositionEdge,
) {
  return edge === "start" ? 0 : composition.cards.length;
}

export type DraftValidationResult = {
  invalidCompositionIds: Set<string>;
  invalidEntryKeys: Set<string>;
};

export function validateDraftedCompositions(
  activeCompositions: CompositionSnapshot[],
  drafts: DraftedCompositionView[],
): DraftValidationResult {
  const invalidCompositionIds = new Set<string>();
  const invalidEntryKeys = new Set<string>();

  for (const draft of drafts) {
    if (draft.tableIndex === null) {
      if (!isValidDraftComposition(draft.entries.map((entry) => entry.card))) {
        invalidCompositionIds.add(draft.id);
      }
      continue;
    }

    const composition = activeCompositions[draft.tableIndex];
    if (!composition) {
      draft.entries.forEach((entry) => invalidEntryKeys.add(entry.key));
      continue;
    }

    const finalCards = cardsWithTableDraft(composition, planTableDraft(composition, draft));
    if (isValidDraftComposition(finalCards, composition.type)) {
      continue;
    }

    const individuallyResponsibleEntries = draft.entries.filter((entry) =>
      isValidDraftComposition(
        cardsWithTableDraft(
          composition,
          planTableDraft(composition, {
            ...draft,
            entries: draft.entries.filter((candidate) => candidate.key !== entry.key),
          }),
        ),
        composition.type,
      ),
    );
    const entriesToMark =
      individuallyResponsibleEntries.length > 0 ? individuallyResponsibleEntries : draft.entries;
    entriesToMark.forEach((entry) => invalidEntryKeys.add(entry.key));
  }

  return { invalidCompositionIds, invalidEntryKeys };
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

    const plan = planTableDraft(activeCompositions[composition.tableIndex], composition);
    for (const addition of plan.additions) {
      additions.push({
        compositionIndex: composition.tableIndex,
        insertIndex: addition.insertIndex,
        cards: addition.entries.map((entry) => entry.card),
      });
    }

    for (const reclaim of plan.reclaims) {
      reclaims.push({
        compositionIndex: composition.tableIndex,
        jokerIndex: reclaim.jokerIndex,
        replacementCard: reclaim.replacementEntry.card,
      });
    }
  }

  return {
    compositions,
    additions,
    reclaims,
  };
}

export function validateOpeningTablePlay(
  playerHasOpened: boolean,
  draftedCompositionsView: DraftedCompositionView[],
): OpeningTablePlayValidation {
  if (playerHasOpened) {
    return { canSubmit: true };
  }

  const hasOwnComposition = draftedCompositionsView.some(
    (composition) =>
      composition.tableIndex === null &&
      composition.entries.length > 0 &&
      composition.entries.every((entry) => !entry.isVirtual),
  );

  return hasOwnComposition
    ? { canSubmit: true }
    : { canSubmit: false, reason: "missing-own-composition" };
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
  }));

  for (const composition of draftCompositions) {
    if (composition.tableIndex === null) {
      continue;
    }

    const existing = views[composition.tableIndex];
    if (existing) {
      const entries = mapHandKeysToEntries(composition.handKeys, entryByKey);
      if (entries.length > 0) {
        existing.draft = planTableDraft(existing.snapshot, { ...composition, entries });
      }
    }
  }

  return views;
}

export function resolveDraftViews(
  activeCompositions: CompositionSnapshot[],
  scopedDraftCompositions: DraftComposition[],
  handEntries: HandEntry[],
) {
  const validHandKeys = new Set(handEntries.map((entry) => entry.key));
  const draftCompositionsFromHand = pruneDraftCompositions(scopedDraftCompositions, validHandKeys);
  const entryByKey = new Map(handEntries.map((entry) => [entry.key, entry]));
  const tableCompositions = buildTableCompositionViews(
    activeCompositions,
    draftCompositionsFromHand,
    entryByKey,
  );
  const virtualReclaimedJokers = buildVirtualReclaimedJokers(tableCompositions);
  const allHandEntries = [...handEntries, ...virtualReclaimedJokers.map((joker) => joker.entry)];
  const allValidHandKeys = new Set(allHandEntries.map((entry) => entry.key));
  const draftCompositions = pruneDraftCompositions(scopedDraftCompositions, allValidHandKeys);
  const allEntryByKey = new Map(allHandEntries.map((entry) => [entry.key, entry]));
  const draftedCompositionsView: DraftedCompositionView[] = draftCompositions.map(
    (composition) => ({
      ...composition,
      entries: mapHandKeysToEntries(composition.handKeys, allEntryByKey),
    }),
  );

  return {
    draftedCompositionsView,
    tableCompositions: buildTableCompositionViews(
      activeCompositions,
      draftCompositions,
      allEntryByKey,
    ),
    virtualReclaimedJokers,
    draftCompositions,
    allHandEntries,
    allEntryByKey,
  };
}

export function buildVirtualReclaimedJokers(tableCompositions: TableCompositionView[]) {
  const virtualJokers: VirtualReclaimedJoker[] = [];

  for (const composition of tableCompositions) {
    for (const reclaim of composition.draft?.reclaims ?? []) {
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

import { arrayMove } from "@dnd-kit/sortable";
import { type CardSnapshot } from "#/components/game-websocket-provider";

export type HandEntry = {
  key: string;
  card: CardSnapshot;
  sourceIndex: number;
};

export type ActiveDrag =
  | { type: "draw"; card: CardSnapshot | null; revealedHandKey: string | null }
  | { type: "hand"; handKey: string };

export type DraftComposition = {
  id: string;
  handKeys: string[];
};

export type DraftedCompositionView = DraftComposition & {
  entries: HandEntry[];
};

export const FACE_DOWN_CARD: CardSnapshot = {};
export const HAND_DROP_ID = "hand-drop-zone";
export const NEW_COMPOSITION_DROP_ID = "new-composition-drop-zone";

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
  const ordered = current
    .map((entry) => nextByKey.get(entry.key) ?? null)
    .filter((entry): entry is HandEntry => entry !== null);
  const seenKeys = new Set(ordered.map((entry) => entry.key));

  return [...ordered, ...next.filter((entry) => !seenKeys.has(entry.key))];
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
  return `draft-composition-${compositionId}`;
}

export function compositionIdFromDropId(dropId: string) {
  if (!dropId.startsWith("draft-composition-")) {
    return null;
  }

  return dropId.slice("draft-composition-".length);
}

export function removeHandKeyFromDrafts(compositions: DraftComposition[], handKey: string) {
  return compositions
    .map((composition) => ({
      ...composition,
      handKeys: composition.handKeys.filter((key) => key !== handKey),
    }))
    .filter((composition) => composition.handKeys.length > 0);
}

export function insertHandKeyIntoDraft(
  compositions: DraftComposition[],
  handKey: string,
  targetCompositionId: string,
  overHandKey?: string,
) {
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

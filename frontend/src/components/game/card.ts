import { type CardSnapshot } from "#/lib/game-protocol";

export const FACE_DOWN_CARD: CardSnapshot = {};

const rankLabels: Record<number, string> = {
  1: "A",
  2: "2",
  3: "3",
  4: "4",
  5: "5",
  6: "6",
  7: "7",
  8: "8",
  9: "9",
  10: "10",
  11: "J",
  12: "Q",
  13: "K",
};

export function rankLabel(rank: number | undefined) {
  return rank === undefined ? undefined : rankLabels[rank];
}

export function cardsEqual(left: CardSnapshot, right: CardSnapshot) {
  return (
    Boolean(left.isJoker) === Boolean(right.isJoker) &&
    (left.rank ?? null) === (right.rank ?? null) &&
    (left.suit ?? null) === (right.suit ?? null)
  );
}

export function keyedCards(cards: CardSnapshot[]) {
  const occurrences = new Map<string, number>();

  return cards.map((card, index) => {
    const identity = card.isJoker ? "joker" : `${card.rank ?? "?"}-${card.suit ?? "?"}`;
    const occurrence = (occurrences.get(identity) ?? 0) + 1;
    occurrences.set(identity, occurrence);
    return { card, index, key: `${identity}-${occurrence}` };
  });
}

export type HandEntry = {
  key: string;
  card: CardSnapshot;
  sourceIndex: number;
  isVirtual?: boolean;
};

export function buildHandEntries(hand: CardSnapshot[]): HandEntry[] {
  return keyedCards(hand).map(({ card, index, key }) => ({ card, key, sourceIndex: index }));
}

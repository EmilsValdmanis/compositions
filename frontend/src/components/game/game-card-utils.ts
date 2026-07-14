import { type CardSnapshot, type CompositionSnapshot } from "#/components/game-websocket-provider";

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

const suitNames: Record<number, string> = {
  0: "Hearts",
  1: "Diamonds",
  2: "Clubs",
  3: "Spades",
};

export function cardName(card: CardSnapshot) {
  if (card.isJoker) {
    return "Joker";
  }

  const rank = rankLabels[card.rank ?? 0] ?? "Unknown";
  const suit = suitNames[card.suit ?? -1] ?? "Unknown";
  return `${rank} of ${suit}`;
}

export function cardPointValue(card: CardSnapshot) {
  if (card.isJoker) {
    return 20;
  }

  const rank = card.rank ?? 0;
  if (rank === 1 || (rank >= 11 && rank <= 13)) {
    return 10;
  }

  if (rank >= 2 && rank <= 10) {
    return rank;
  }

  return 0;
}

export function cardPointTotal(cards: CardSnapshot[]) {
  return cards.reduce((total, card) => total + cardPointValue(card), 0);
}

function rankPointValue(rank: number, aceLow = false) {
  if (rank === 1) {
    return aceLow ? 1 : 10;
  }

  if (rank >= 11 && rank <= 13) {
    return 10;
  }

  if (rank >= 2 && rank <= 10) {
    return rank;
  }

  return 0;
}

function draftSetPointTotal(cards: CardSnapshot[]) {
  if (cards.length < 3 || cards.length > 4) {
    return null;
  }

  const naturalCards = cards.filter((card) => !card.isJoker);
  const setRank = naturalCards[0]?.rank ?? 1;
  const seenSuits = new Set<number>();

  for (const card of naturalCards) {
    if (card.rank !== setRank || typeof card.suit !== "number" || seenSuits.has(card.suit)) {
      return null;
    }
    seenSuits.add(card.suit);
  }

  return cards.length * rankPointValue(setRank);
}

function draftRunPointTotal(cards: CardSnapshot[]) {
  if (cards.length < 3 || cards.length > 14) {
    return null;
  }

  const naturalCards = cards.filter((card) => !card.isJoker);
  const runSuit = naturalCards[0]?.suit;

  if (naturalCards.some((card) => card.suit !== runSuit || typeof card.rank !== "number")) {
    return null;
  }

  let best: number | null = null;
  const runLength = cards.length;

  for (let start = 1; start <= 15 - runLength; start += 1) {
    const end = start + runLength - 1;
    const usedPositions = new Set<number>();
    let fits = true;

    for (const card of naturalCards) {
      const rank = card.rank ?? 0;
      const possiblePositions = rank === 1 ? [1, 14] : [rank];
      const position = possiblePositions.find((candidate) => {
        return candidate >= start && candidate <= end && !usedPositions.has(candidate);
      });

      if (position === undefined) {
        fits = false;
        break;
      }

      usedPositions.add(position);
    }

    if (!fits) {
      continue;
    }

    let total = 0;
    for (let rank = start; rank <= end; rank += 1) {
      total += rankPointValue(rank === 14 ? 1 : rank, rank === 1);
    }

    best = Math.max(best ?? 0, total);
  }

  return best;
}

export function draftCompositionPointTotal(cards: CardSnapshot[]) {
  const validTotals = [draftSetPointTotal(cards), draftRunPointTotal(cards)].filter(
    (total): total is number => typeof total === "number",
  );

  if (validTotals.length > 0) {
    return Math.max(...validTotals);
  }

  // A joker's 20-point value only applies while it is left in a player's hand.
  // Until a natural card gives an unfinished draft some composition context,
  // its table value cannot be known.
  return cards.length > 0 && cards.every((card) => card.isJoker) ? null : cardPointTotal(cards);
}

export function isValidDraftComposition(cards: CardSnapshot[], type?: CompositionSnapshot["type"]) {
  if (type === "set") {
    return draftSetPointTotal(cards) !== null;
  }

  if (type === "run") {
    return draftRunPointTotal(cards) !== null;
  }

  return draftSetPointTotal(cards) !== null || draftRunPointTotal(cards) !== null;
}

export function draftCompositionPreviewPointTotal(
  composition: CompositionSnapshot,
  additions: CardSnapshot[],
  replacements: Array<{ jokerIndex: number; replacementCard: CardSnapshot }> = [],
) {
  const replacementByJokerIndex = new Map(
    replacements.map((replacement) => [replacement.jokerIndex, replacement.replacementCard]),
  );
  const previewCards = composition.cards.map(
    (card, index) => replacementByJokerIndex.get(index) ?? card,
  );

  return draftCompositionPointTotal([...previewCards, ...additions]);
}

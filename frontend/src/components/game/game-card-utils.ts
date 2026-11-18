import { rankLabel } from "#/components/game/card";
import { type CardSnapshot, type CompositionSnapshot } from "#/lib/game-protocol";
import { m } from "#/paraglide/messages.js";

const suitNames: Record<number, () => string> = {
  0: m.card_hearts,
  1: m.card_diamonds,
  2: m.card_clubs,
  3: m.card_spades,
};

export function cardName(card: CardSnapshot) {
  if (card.isJoker) {
    return m.joker();
  }

  const rank = rankLabel(card.rank) ?? m.unknown();
  const suit = suitNames[card.suit ?? -1]?.() ?? m.unknown();
  return m.card_name({ rank, suit });
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

export function jokerReclaimPointValue(
  composition: CompositionSnapshot,
  jokerIndex: number,
  replacementCard: CardSnapshot,
) {
  if (!composition.cards[jokerIndex]?.isJoker || typeof replacementCard.rank !== "number") {
    return null;
  }

  const nextCard = composition.cards[jokerIndex + 1];
  const nextRepresentation = composition.jokerRepresentations?.[jokerIndex + 1];
  const nextRank = nextCard?.isJoker ? nextRepresentation?.[0]?.rank : nextCard?.rank;
  const aceLow =
    composition.type === "run" && jokerIndex === 0 && replacementCard.rank === 1 && nextRank === 2;

  return rankPointValue(replacementCard.rank, aceLow);
}

function hasValidNaturalIdentity(card: CardSnapshot) {
  return (
    card.isJoker === true ||
    (Number.isInteger(card.rank) &&
      (card.rank ?? 0) >= 1 &&
      (card.rank ?? 0) <= 13 &&
      Number.isInteger(card.suit) &&
      (card.suit ?? -1) >= 0 &&
      (card.suit ?? -1) <= 3)
  );
}

function draftSetPointTotal(cards: CardSnapshot[]) {
  if (
    cards.length < 3 ||
    cards.length > 4 ||
    cards.some((card) => !hasValidNaturalIdentity(card))
  ) {
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
  if (
    cards.length < 3 ||
    cards.length > 14 ||
    cards.some((card) => !hasValidNaturalIdentity(card))
  ) {
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

export function draftCompositionPointTotal(
  cards: CardSnapshot[],
  type?: CompositionSnapshot["type"],
) {
  if (cards.every((card) => card.isJoker)) {
    return null;
  }

  const resolvedPoints =
    type === "set"
      ? draftSetPointTotal(cards)
      : type === "run"
        ? draftRunPointTotal(cards)
        : // Match the backend's composition inference order: a valid set wins before
          // the same cards are considered as an unordered run.
          (draftSetPointTotal(cards) ?? draftRunPointTotal(cards));

  if (resolvedPoints !== null) {
    return resolvedPoints;
  }

  if (cards.some((card) => card.isJoker) || cards.some((card) => !hasValidNaturalIdentity(card))) {
    return null;
  }

  return cards.reduce((total, card) => total + rankPointValue(card.rank ?? 0), 0);
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

export function isCompleteDraftComposition(
  cards: CardSnapshot[],
  type?: CompositionSnapshot["type"],
) {
  if (cards.some((card) => card.isJoker)) {
    return false;
  }

  if (type === "set" || (type === undefined && draftSetPointTotal(cards) !== null)) {
    return cards.length === 4 && draftSetPointTotal(cards) !== null;
  }

  return cards.length === 14 && draftRunPointTotal(cards) !== null;
}

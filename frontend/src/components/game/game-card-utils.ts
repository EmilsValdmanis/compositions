import { type CardSnapshot } from "#/components/game-websocket-provider";

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

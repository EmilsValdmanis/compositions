import { type CardSnapshot } from "#/components/game-websocket-provider";

export function cardsEqual(left: CardSnapshot, right: CardSnapshot) {
  return (
    Boolean(left.isJoker) === Boolean(right.isJoker) &&
    left.rank === right.rank &&
    left.suit === right.suit
  );
}

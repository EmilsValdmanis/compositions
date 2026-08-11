import { type CardSnapshot, type GameSnapshot } from "#/components/game-websocket-provider";
import { FACE_DOWN_CARD } from "#/components/game/game-board-view-state";

export type CardTransfer = {
  actorPlayerId: string;
  card: CardSnapshot;
  faceDown: boolean;
  source: "deck" | "discard" | "player";
  target: "discard" | "player";
};

export type CompletedCompositionCollection = {
  actorPlayerId: string;
  collectedCards: CardSnapshot[];
  discardCard: CardSnapshot;
  previousTopDiscard: CardSnapshot | null;
};

function cardsEqual(left: CardSnapshot, right: CardSnapshot) {
  return (
    Boolean(left.isJoker) === Boolean(right.isJoker) &&
    left.rank === right.rank &&
    left.suit === right.suit
  );
}

function cardListsEqual(left: CardSnapshot[], right: CardSnapshot[]) {
  return (
    left.length === right.length && left.every((card, index) => cardsEqual(card, right[index]))
  );
}

function playerHandCount(game: GameSnapshot, playerId: string) {
  return game.players.find((player) => player.playerId === playerId)?.handCount;
}

function sameTurn(previous: GameSnapshot, current: GameSnapshot) {
  return (
    previous.round === current.round &&
    previous.turn.number === current.turn.number &&
    previous.turn.playerId === current.turn.playerId
  );
}

export function inferCompletedCompositionCollection(
  previous: GameSnapshot,
  current: GameSnapshot,
): CompletedCompositionCollection | null {
  const addedCardCount = current.discardPile.length - previous.discardPile.length;
  const discardCard = current.discardPile[0];
  const actorPlayerId = previous.turn.playerId;

  if (!actorPlayerId || !discardCard || addedCardCount <= 1) {
    return null;
  }

  if (!cardListsEqual(current.discardPile.slice(addedCardCount), previous.discardPile)) {
    return null;
  }

  return {
    actorPlayerId,
    collectedCards: current.discardPile.slice(1, addedCardCount),
    discardCard,
    previousTopDiscard: previous.discardPile[0] ?? null,
  };
}

export function inferCardTransfer(
  previous: GameSnapshot,
  current: GameSnapshot,
  viewerPlayerId: string,
): CardTransfer | null {
  const previousActorId = previous.turn.playerId;
  const currentActorId = current.turn.playerId;

  if (
    currentActorId &&
    currentActorId !== viewerPlayerId &&
    sameTurn(previous, current) &&
    !previous.turn.hasDrawn &&
    current.turn.hasDrawn &&
    playerHandCount(current, currentActorId) ===
      (playerHandCount(previous, currentActorId) ?? 0) + 1
  ) {
    const drawSource =
      current.turnActivity?.drawSource ??
      (current.drawPileCount < previous.drawPileCount ? "deck" : "discard");

    if (drawSource === "discard") {
      const card = previous.discardPile[0];
      return card
        ? {
            actorPlayerId: currentActorId,
            card,
            faceDown: false,
            source: "discard",
            target: "player",
          }
        : null;
    }

    return {
      actorPlayerId: currentActorId,
      card: FACE_DOWN_CARD,
      faceDown: true,
      source: "deck",
      target: "player",
    };
  }

  const placedDiscard = current.discardPile[0];
  if (
    previousActorId &&
    previousActorId !== viewerPlayerId &&
    !sameTurn(previous, current) &&
    placedDiscard &&
    current.discardPile.length > previous.discardPile.length
  ) {
    return {
      actorPlayerId: previousActorId,
      card: placedDiscard,
      faceDown: false,
      source: "player",
      target: "discard",
    };
  }

  return null;
}

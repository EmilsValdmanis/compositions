import { useMemo, useState } from "react";
import { mockScenarios } from "#/dev/mock-game-scenarios";
import { GameBoardView } from "#/components/game/game-board-view";
import { playerName } from "#/components/game/game-view-helpers";
import {
  type ActionResult,
  type CardSnapshot,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type RoomSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";

const scenarios = mockScenarios;

function cloneCards(cards: CardSnapshot[]) {
  return cards.map((card) => ({ ...card }));
}

function cloneDrafts(drafts: DraftCompositionSnapshot[] | undefined) {
  return drafts?.map((draft) => ({
    ...draft,
    cards: cloneCards(draft.cards),
  }));
}

function cloneGame(game: GameSnapshot): GameSnapshot {
  return {
    ...game,
    turn: { ...game.turn },
    players: game.players.map((player) => ({ ...player })),
    hand: cloneCards(game.hand),
    discardPile: cloneCards(game.discardPile),
    activeCompositions: game.activeCompositions.map((composition) => ({
      ...composition,
      cards: cloneCards(composition.cards),
      jokerRepresentations: composition.jokerRepresentations
        ? Object.fromEntries(
            Object.entries(composition.jokerRepresentations).map(([key, cards]) => [
              key,
              cloneCards(cards),
            ]),
          )
        : undefined,
    })),
    turnActivity: game.turnActivity
      ? {
          ...game.turnActivity,
          baselineCompositions: game.turnActivity.baselineCompositions?.map((composition) => ({
            ...composition,
            cards: cloneCards(composition.cards),
            jokerRepresentations: composition.jokerRepresentations
              ? Object.fromEntries(
                  Object.entries(composition.jokerRepresentations).map(([key, cards]) => [
                    key,
                    cloneCards(cards),
                  ]),
                )
              : undefined,
          })),
          draftCompositions: cloneDrafts(game.turnActivity.draftCompositions),
          compositionActivities: game.turnActivity.compositionActivities?.map((activity) => ({
            ...activity,
            cardActivities: activity.cardActivities ? { ...activity.cardActivities } : undefined,
          })),
        }
      : undefined,
  };
}

function cloneRoom(room: RoomSnapshot): RoomSnapshot {
  return {
    ...room,
    pendingDealChoice: room.pendingDealChoice ? { ...room.pendingDealChoice } : undefined,
    players: room.players.map((player) => ({ ...player })),
  };
}

function drawFromDeck(game: GameSnapshot) {
  if (game.drawPileCount <= 0 || game.turn.hasDrawn) {
    return game;
  }

  const nextRank = ((game.turn.number + game.round + game.hand.length) % 13) + 1;
  const nextSuit = (game.turn.playerIndex + game.hand.length) % 4;

  return {
    ...game,
    hand: [...game.hand, { rank: nextRank, suit: nextSuit }],
    drawPileCount: Math.max(0, game.drawPileCount - 1),
    turn: {
      ...game.turn,
      hasDrawn: true,
    },
  };
}

function drawFromDiscard(game: GameSnapshot) {
  if (game.turn.hasDrawn || game.discardPile.length === 0) {
    return game;
  }

  const [topDiscard, ...remainingDiscard] = game.discardPile;

  return {
    ...game,
    hand: [...game.hand, topDiscard],
    discardPile: remainingDiscard,
    turn: {
      ...game.turn,
      hasDrawn: true,
      mustUseDiscardDraw: false,
    },
  };
}

function discardFromHand(game: GameSnapshot, cardIndex: number) {
  if (!game.turn.hasDrawn || cardIndex < 0 || cardIndex >= game.hand.length) {
    return game;
  }

  const nextHand = [...game.hand];
  const [discardedCard] = nextHand.splice(cardIndex, 1);
  const nextPlayerIndex = (game.turn.playerIndex + 1) % game.players.length;
  const nextPlayer = game.players[nextPlayerIndex];

  return {
    ...game,
    hand: nextHand,
    discardPile: discardedCard ? [discardedCard, ...game.discardPile] : game.discardPile,
    turn: {
      ...game.turn,
      number: game.turn.number + 1,
      playerIndex: nextPlayerIndex,
      playerId: nextPlayer?.playerId,
      hasDrawn: false,
      mustUseDiscardDraw: false,
    },
  };
}

function applyTablePlay(game: GameSnapshot, play: TablePlayRequest) {
  const nextHand = [...game.hand];

  function removeCard(card: CardSnapshot) {
    const index = nextHand.findIndex(
      (item) =>
        Boolean(item.isJoker) === Boolean(card.isJoker) &&
        item.rank === card.rank &&
        item.suit === card.suit,
    );

    if (index >= 0) {
      nextHand.splice(index, 1);
    }
  }

  const nextCompositions = game.activeCompositions.map((composition) => ({
    ...composition,
    cards: cloneCards(composition.cards),
    jokerRepresentations: composition.jokerRepresentations
      ? Object.fromEntries(
          Object.entries(composition.jokerRepresentations).map(([key, cards]) => [
            key,
            cloneCards(cards),
          ]),
        )
      : undefined,
  }));

  for (const composition of play.compositions) {
    composition.cards.forEach(removeCard);
    nextCompositions.push({
      type: "run",
      cards: cloneCards(composition.cards),
      points: composition.cards.length * 10,
      complete: composition.cards.length >= 3,
      jokerRepresentations: undefined,
    });
  }

  for (const addition of play.additions) {
    addition.cards.forEach(removeCard);
    const target = nextCompositions[addition.compositionIndex];
    if (target) {
      const insertIndex = addition.insertIndex ?? target.cards.length;
      target.cards.splice(insertIndex, 0, ...cloneCards(addition.cards));
      target.points += addition.cards.length * 5;
      target.complete = target.cards.length >= 3;
    }
  }

  for (const reclaim of play.reclaims) {
    removeCard(reclaim.replacementCard);
    const target = nextCompositions[reclaim.compositionIndex];
    if (!target || !target.cards[reclaim.jokerIndex]?.isJoker) {
      continue;
    }

    target.jokerRepresentations = {
      ...target.jokerRepresentations,
      [reclaim.jokerIndex]: [reclaim.replacementCard],
    };
  }

  return {
    ...game,
    hand: nextHand,
    activeCompositions: nextCompositions,
  };
}

export function DevGameUi() {
  const scenario = scenarios[0];
  const [gameOverride, setGameOverride] = useState<GameSnapshot | null>(null);

  const players = scenario?.players ?? [];
  const room = scenario ? cloneRoom(scenario.room) : null;
  const rawGame = useMemo(
    () => (gameOverride ? cloneGame(gameOverride) : scenario ? cloneGame(scenario.game) : null),
    [gameOverride, scenario],
  );
  const game = rawGame;

  const resolvedPerspectiveId = scenario?.controlledPlayerId ?? "";

  const connectedPlayers = players.filter((player) => player.connected).length;
  const isMyTurn = game?.turn.playerId === resolvedPerspectiveId;
  const canDraw = Boolean(game) && isMyTurn && !game.turn.hasDrawn;
  const topDiscardCard = game?.discardPile[0] ?? null;
  const canDrawDeck = canDraw && !game?.turn.mustUseDiscardDraw && (game?.drawPileCount ?? 0) > 0;
  const canDrawDiscard = canDraw && Boolean(topDiscardCard);
  const canDiscard = Boolean(game) && isMyTurn && Boolean(game.turn.hasDrawn);
  const turnPlayerName = playerName(players, game?.turn.playerId);

  function updateGame(updater: (current: GameSnapshot) => GameSnapshot) {
    setGameOverride((current) => {
      const baseGame = current ? cloneGame(current) : scenario ? cloneGame(scenario.game) : null;
      return baseGame ? updater(baseGame) : null;
    });
  }

  if (!import.meta.env.DEV || !scenario || !game || !room) {
    return null;
  }

  async function handleDiscardCard(cardIndex: number) {
    updateGame((current) => discardFromHand(current, cardIndex));

    return {
      action: "discard",
      playerId: resolvedPerspectiveId,
      ok: true,
    } satisfies ActionResult;
  }

  async function handlePlayTable(play: TablePlayRequest) {
    updateGame((current) => applyTablePlay(current, play));

    return {
      action: "play",
      playerId: resolvedPerspectiveId,
      ok: true,
    } satisfies ActionResult;
  }

  return (
    <section className="mx-auto flex h-full min-h-0 w-full flex-1 flex-col gap-3 md:gap-4">
      <div className="flex min-h-0 flex-1 flex-col overflow-visible">
        <GameBoardView
          game={game}
          roomCode={room.code}
          playerId={resolvedPerspectiveId}
          players={players}
          connectedPlayers={connectedPlayers}
          turnState={{
            canDrawDeck,
            canDrawDiscard,
            canDiscard,
            isMyTurn,
            turnPlayerName,
          }}
          topDiscardCard={topDiscardCard}
          onDiscardCard={handleDiscardCard}
          onDrawFromDeck={() => updateGame(drawFromDeck)}
          onDrawFromDiscard={() => updateGame(drawFromDiscard)}
          onPlayTable={handlePlayTable}
          disableDraftSync
        />
      </div>
    </section>
  );
}

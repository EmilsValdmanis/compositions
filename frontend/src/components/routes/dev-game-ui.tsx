import { useState } from "react";
import { mockScenarios } from "#/dev/mock-game-scenarios";
import { GameBoardView } from "#/components/game/game-board-view";
import { GameCard } from "#/components/game/game-card";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { GameResultsView } from "#/components/game/game-results-view";
import { playerName } from "#/components/game/game-view-helpers";
import { Tabs, TabsList, TabsTrigger } from "#/components/ui/tabs";
import {
  type ActionResult,
  type CardSnapshot,
  type DealingChoiceRequest,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type RoomSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";
import { m } from "#/paraglide/messages.js";

const scenarios = mockScenarios;
type DevViewMode = "start" | "board" | "deal" | "results" | "cards";

const deckRanks = [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 1] as const;
const deckSuits = [3, 0, 2, 1] as const;

function CardDeckGallery() {
  return (
    <div className="flex overflow-auto p-3 md:p-5">
      <div className="m-auto flex w-max min-w-full flex-col gap-2.5">
        {deckSuits.map((suit) => (
          <div key={suit} className="flex w-max gap-2">
            {deckRanks.map((rank) => (
              <GameCard key={`${rank}-${suit}`} card={{ rank, suit }} />
            ))}
          </div>
        ))}
        <div className="flex w-max gap-2 ">
          <GameCard card={{ isJoker: true }} />
          <GameCard card={{ isJoker: true }} />
        </div>
      </div>
    </div>
  );
}

function handleChooseDealing(_choice: DealingChoiceRequest | string) {}

function cloneCards(cards: CardSnapshot[]) {
  return cards.map((card) => ({ ...card }));
}

function cloneDrafts(drafts: DraftCompositionSnapshot[] | undefined) {
  return drafts?.map((draft) => ({
    ...draft,
    cardPlacements: draft.cardPlacements?.map((placement) => ({ ...placement })),
    cards: cloneCards(draft.cards),
  }));
}

function cloneGame(game: GameSnapshot): GameSnapshot {
  return {
    ...game,
    turn: { ...game.turn },
    players: game.players.map((player) => ({
      ...player,
      hand: player.hand ? cloneCards(player.hand) : undefined,
    })),
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

const leftoverHandsByPlayerId: Record<string, CardSnapshot[]> = {
  "player-avery": [],
  "player-blair": [
    { rank: 3, suit: 0 },
    { rank: 6, suit: 2 },
    { rank: 11, suit: 3 },
    { isJoker: true },
    { rank: 13, suit: 0 },
    { rank: 1, suit: 1 },
  ],
  "player-casey": [
    { rank: 2, suit: 3 },
    { rank: 7, suit: 0 },
    { rank: 10, suit: 1 },
    { rank: 12, suit: 2 },
    { rank: 4, suit: 0 },
    { rank: 8, suit: 3 },
    { rank: 9, suit: 1 },
    { rank: 5, suit: 2 },
    { rank: 6, suit: 0 },
  ],
  "player-devon": [
    { rank: 1, suit: 0 },
    { rank: 2, suit: 1 },
    { rank: 3, suit: 2 },
    { rank: 4, suit: 3 },
    { rank: 5, suit: 0 },
    { rank: 6, suit: 1 },
    { rank: 7, suit: 2 },
    { rank: 8, suit: 0 },
    { rank: 9, suit: 3 },
    { rank: 10, suit: 0 },
    { rank: 11, suit: 1 },
  ],
};

function revealLeftoverHands(game: GameSnapshot): GameSnapshot {
  return {
    ...game,
    players: game.players.map((player) => {
      const hand = leftoverHandsByPlayerId[player.playerId];

      if (!hand) {
        return player;
      }

      return {
        ...player,
        hand: cloneCards(hand),
        handCount: hand.length,
      };
    }),
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

function cardsMatch(left: CardSnapshot, right: CardSnapshot) {
  return (
    Boolean(left.isJoker) === Boolean(right.isJoker) &&
    left.rank === right.rank &&
    left.suit === right.suit
  );
}

function discardFromHand(game: GameSnapshot, cardIndex: number, expectedCard: CardSnapshot) {
  if (!game.turn.hasDrawn) {
    return game;
  }

  if (!game.hand[cardIndex] || !cardsMatch(game.hand[cardIndex], expectedCard)) {
    cardIndex = game.hand.findIndex((card) => cardsMatch(card, expectedCard));
  }
  if (cardIndex < 0) {
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
  const [viewMode, setViewMode] = useState<DevViewMode>("board");
  const [lobbyRoom, setLobbyRoom] = useState<RoomSnapshot | null>(null);
  const [lobbyRoomCode, setLobbyRoomCode] = useState("");

  const players = scenario?.players ?? [];
  const room = scenario ? cloneRoom(scenario.room) : null;
  const rawGame = gameOverride
    ? cloneGame(gameOverride)
    : scenario
      ? cloneGame(scenario.game)
      : null;
  const game = rawGame;
  const resultsGame = rawGame ? revealLeftoverHands(rawGame) : null;

  const resolvedPerspectiveId = scenario?.controlledPlayerId ?? "";

  const connectedPlayers = players.filter((player) => player.connected).length;
  const isMyTurn = game?.turn.playerId === resolvedPerspectiveId;
  const canDraw = Boolean(game) && isMyTurn && !game.turn.hasDrawn;
  const topDiscardCard = game?.discardPile[0] ?? null;
  const canDrawDeck = canDraw && !game?.turn.mustUseDiscardDraw && (game?.drawPileCount ?? 0) > 0;
  const canDrawDiscard = canDraw && Boolean(topDiscardCard);
  const canDiscard = Boolean(game) && isMyTurn && Boolean(game.turn.hasDrawn);
  const turnPlayerName = playerName(players, game?.turn.playerId);
  const resultsRoom = room
    ? {
        ...room,
        phase: "round_over",
        pendingDealChoice: undefined,
      }
    : null;

  function updateGame(updater: (current: GameSnapshot) => GameSnapshot) {
    setGameOverride((current) => {
      const baseGame = current ? cloneGame(current) : scenario ? cloneGame(scenario.game) : null;
      return baseGame ? updater(baseGame) : null;
    });
  }

  if (!import.meta.env.DEV || !scenario || !game || !room) {
    return null;
  }

  async function handleDiscardCard(cardIndex: number, card: CardSnapshot) {
    updateGame((current) => discardFromHand(current, cardIndex, card));

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

  async function handlePlayTableAndDiscard(
    play: TablePlayRequest,
    cardIndex: number,
    card: CardSnapshot,
  ) {
    updateGame((current) => discardFromHand(applyTablePlay(current, play), cardIndex, card));

    return {
      action: "play_and_discard",
      playerId: resolvedPerspectiveId,
      ok: true,
    } satisfies ActionResult;
  }

  function handleStartNextRound() {
    setGameOverride(null);
    setViewMode("board");
  }

  function enterLobbyRoom(code: string) {
    if (!room) return;

    setLobbyRoom({
      ...cloneRoom(room),
      code: code.trim().toUpperCase() || room.code,
      phase: "lobby",
      pendingDealChoice: undefined,
    });
    setLobbyRoomCode("");
  }

  return (
    <section className="mx-auto flex h-full min-h-0 w-full flex-1 flex-col gap-2 md:gap-4 [@media(max-height:600px)]:gap-2">
      <div className="flex shrink-0 items-center justify-end">
        <Tabs
          value={viewMode}
          onValueChange={(value) => setViewMode(value as DevViewMode)}
          className="flex-none"
        >
          <TabsList aria-label={m.dev_preview_mode()}>
            <TabsTrigger value="start">{m.start()}</TabsTrigger>
            <TabsTrigger value="board">{m.board()}</TabsTrigger>
            <TabsTrigger value="deal">{m.deal()}</TabsTrigger>
            <TabsTrigger value="results">{m.results()}</TabsTrigger>
            <TabsTrigger value="cards">{m.cards()}</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden md:overflow-visible [@media(max-height:600px)]:overflow-hidden">
        {viewMode === "cards" ? (
          <CardDeckGallery />
        ) : viewMode === "start" ? (
          <div className="flex min-h-0 flex-1 overflow-auto">
            <GameLobbyView
              room={lobbyRoom}
              game={null}
              completedGame={null}
              players={lobbyRoom?.players ?? []}
              roomCode={lobbyRoomCode}
              roomActions={{
                canCreateRoom: lobbyRoom == null,
                canJoinRoom: lobbyRoom == null && lobbyRoomCode.trim().length > 0,
                canLeaveRoom: lobbyRoom != null,
                canStartGame: lobbyRoom != null,
              }}
              dealChoice={{
                pendingDealChoice: null,
                dealChooserName: null,
                isDealChooser: false,
              }}
              onRoomCodeChange={setLobbyRoomCode}
              onCreateRoom={() => enterLobbyRoom(room.code)}
              onJoinRoom={enterLobbyRoom}
              onStartGame={() => setViewMode("board")}
              onChooseDealing={handleChooseDealing}
              onLeaveRoom={() => setLobbyRoom(null)}
              onSendEmote={() => {}}
              onCopyRoomCode={() => {}}
              onCopyRoomLink={() => {}}
            />
          </div>
        ) : viewMode === "deal" ? (
          <div className="mx-auto flex w-full max-w-xl flex-1 items-center px-2 py-6">
            <DealChoicePanel
              players={players}
              pendingDealChoice={{
                dealerIndex: 3,
                chooserIndex: 0,
                chooserPlayerId: players[0]?.playerId ?? "player-avery",
              }}
              dealChooserName={players[0]?.name ?? "Avery"}
              isDealChooser
              onChooseDealing={handleChooseDealing}
            />
          </div>
        ) : viewMode === "results" && resultsGame ? (
          <div className="flex min-h-0 flex-1 overflow-auto">
            <GameResultsView
              room={resultsRoom}
              game={resultsGame}
              players={players}
              playerId={resolvedPerspectiveId}
              connectedPlayers={connectedPlayers}
              dealChoice={{
                pendingDealChoice: null,
                dealChooserName: null,
                isDealChooser: false,
              }}
              onStartNextRound={handleStartNextRound}
              onChooseDealing={handleChooseDealing}
              onSendEmote={() => {}}
            />
          </div>
        ) : (
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
            onPlayTableAndDiscard={handlePlayTableAndDiscard}
            onSendEmote={() => {}}
            draftSyncMode="disabled"
          />
        )}
      </div>
    </section>
  );
}

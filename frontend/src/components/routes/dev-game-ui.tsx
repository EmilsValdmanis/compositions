import { useEffect, useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";
import { mockScenarios, type MockScenario } from "#/dev/mock-game-scenarios";
import { GameBoardHeader } from "#/components/game/game-board-header";
import { GameBoardView } from "#/components/game/game-board-view";
import { playerName } from "#/components/game/game-view-helpers";
import {
  type CardSnapshot,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import { Separator } from "#/components/ui/separator";

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
          Object.entries(composition.jokerRepresentations).map(([key, cards]) => [key, cloneCards(cards)]),
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
      target.cards.push(...cloneCards(addition.cards));
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
      ...(target.jokerRepresentations ?? {}),
      [reclaim.jokerIndex]: [reclaim.replacementCard],
    };
  }

  return {
    ...game,
    hand: nextHand,
    activeCompositions: nextCompositions,
  };
}

function perspectivePlayerIds(players: PlayerSnapshot[], controlledPlayerId: string) {
  const controlledFirst = players.find((player) => player.playerId === controlledPlayerId);
  const others = players.filter((player) => player.playerId !== controlledPlayerId);
  return controlledFirst
    ? [controlledFirst.playerId, ...others.map((player) => player.playerId)]
    : players.map((player) => player.playerId);
}

export function DevGameUi() {
  const [scenarioId, setScenarioId] = useState(scenarios[0]?.id ?? "");
  const scenario = useMemo(
    () => scenarios.find((item) => item.id === scenarioId) ?? scenarios[0],
    [scenarioId],
  );
  const [perspectivePlayerId, setPerspectivePlayerId] = useState(
    scenario?.controlledPlayerId ?? "",
  );
  const [gameOverride, setGameOverride] = useState<GameSnapshot | null>(null);

  const players = scenario?.players ?? [];
  const room = scenario ? cloneRoom(scenario.room) : null;
  const game = useMemo(
    () => (gameOverride ? cloneGame(gameOverride) : scenario ? cloneGame(scenario.game) : null),
    [gameOverride, scenario],
  );

  const availablePerspectiveIds = useMemo(
    () => (scenario ? perspectivePlayerIds(players, scenario.controlledPlayerId) : []),
    [players, scenario],
  );

  useEffect(() => {
    if (!availablePerspectiveIds.includes(perspectivePlayerId)) {
      setPerspectivePlayerId(availablePerspectiveIds[0] ?? "");
    }
  }, [availablePerspectiveIds, perspectivePlayerId]);

  const selectedPerspective =
    players.find((player) => player.playerId === perspectivePlayerId) ?? players[0] ?? null;
  const resolvedPerspectiveId = selectedPerspective?.playerId ?? "";
  const connectedPlayers = players.filter((player) => player.connected).length;
  const phase = room?.phase ?? "in_progress";
  const isMyTurn = game?.turn.playerId === resolvedPerspectiveId;
  const canDraw = Boolean(game) && isMyTurn && !game.turn.hasDrawn;
  const topDiscardCard = game?.discardPile[0] ?? null;
  const canDrawDeck = canDraw && !game?.turn.mustUseDiscardDraw && (game?.drawPileCount ?? 0) > 0;
  const canDrawDiscard = canDraw && Boolean(topDiscardCard);
  const canDiscard = Boolean(game) && isMyTurn && Boolean(game.turn.hasDrawn);
  const turnPlayerName = playerName(players, game?.turn.playerId);

  function resetScenario(nextScenarioId: string, nextPerspectiveId?: string) {
    const nextScenario = scenarios.find((item) => item.id === nextScenarioId) ?? scenarios[0];
    setScenarioId(nextScenario?.id ?? "");
    setPerspectivePlayerId(nextPerspectiveId ?? nextScenario?.controlledPlayerId ?? "");
    setGameOverride(null);
  }

  function updateGame(updater: (current: GameSnapshot) => GameSnapshot) {
    setGameOverride((current) => {
      const baseGame = current ? cloneGame(current) : scenario ? cloneGame(scenario.game) : null;
      return baseGame ? updater(baseGame) : null;
    });
  }

  if (!import.meta.env.DEV || !scenario || !game || !room) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Dev game UI</CardTitle>
          <CardDescription>This page is only available in development.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <section className="mx-auto flex h-full min-h-0 w-full max-w-[1700px] flex-1 flex-col gap-4 overflow-hidden">
      <Card size="sm" className="shadow-sm">
        <CardHeader className="gap-3">
          <div className="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <CardTitle>Dev Game UI</CardTitle>
              <CardDescription>
                Switch scenarios and player perspectives to inspect the board as if a live game were in progress.
              </CardDescription>
            </div>
            <CardAction className="flex items-center gap-2 self-start">
              <Button asChild variant="outline" size="sm">
                <Link to="/">Back to game</Link>
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => resetScenario(scenario.id, resolvedPerspectiveId)}
              >
                Reset mock state
              </Button>
            </CardAction>
          </div>
          <div className="flex flex-wrap gap-2">
            {scenarios.map((item) => (
              <Button
                key={item.id}
                type="button"
                size="sm"
                variant={item.id === scenario.id ? "default" : "outline"}
                onClick={() => resetScenario(item.id)}
              >
                {item.label}
              </Button>
            ))}
          </div>
          <div className="flex flex-wrap gap-2">
            {availablePerspectiveIds.map((playerId) => {
              const player = players.find((item) => item.playerId === playerId);
              if (!player) {
                return null;
              }

              const isActive = player.playerId === resolvedPerspectiveId;

              return (
                <Button
                  key={player.playerId}
                  type="button"
                  size="sm"
                  variant={isActive ? "secondary" : "outline"}
                  onClick={() => setPerspectivePlayerId(player.playerId)}
                >
                  {player.name}
                </Button>
              );
            })}
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 text-sm text-muted-foreground">
          <p>{scenario.description}</p>
          <div className="flex flex-wrap gap-2">
            <Badge variant="outline">Room {room.code}</Badge>
            <Badge variant="outline">Viewing as {selectedPerspective?.name ?? "Unknown"}</Badge>
            <Badge variant={isMyTurn ? "default" : "secondary"}>
              {isMyTurn ? "Active player" : "Spectator perspective"}
            </Badge>
          </div>
        </CardContent>
      </Card>

      <div className="grid min-h-0 flex-1 gap-4 overflow-hidden lg:grid-cols-[20rem_minmax(0,1fr)] xl:grid-cols-[22rem_minmax(0,1fr)]">
        <Card className="min-h-0 overflow-y-auto">
          <CardHeader>
            <CardTitle>Scenario Notes</CardTitle>
            <CardDescription>Quick controls for iterating on the mock turn state.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">Current turn</p>
              <div className="flex flex-wrap gap-2">
                <Badge variant="outline">{turnPlayerName}</Badge>
                <Badge variant="outline">Turn {game.turn.number}</Badge>
                <Badge variant="outline">Round {game.round}</Badge>
                <Badge variant="outline">{game.turn.hasDrawn ? "Has drawn" : "Needs draw"}</Badge>
              </div>
            </div>
            <Separator />
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">Quick actions</p>
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!canDrawDeck}
                  onClick={() => updateGame(drawFromDeck)}
                >
                  Draw from deck
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!canDrawDiscard}
                  onClick={() => updateGame(drawFromDiscard)}
                >
                  Draw discard
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!canDiscard}
                  onClick={() => updateGame((current) => discardFromHand(current, current.hand.length - 1))}
                >
                  Discard last card
                </Button>
              </div>
            </div>
            <Separator />
            <div className="space-y-2 text-sm">
              <p className="font-medium text-foreground">What this scenario shows</p>
              <ul className="space-y-1 text-muted-foreground">
                <li>Joker reclaim highlights</li>
                <li>New composition cards and table activity labels</li>
                <li>Drafted new compositions not yet submitted</li>
                <li>Additions staged onto existing compositions</li>
                <li>Existing table compositions from previous turns</li>
              </ul>
            </div>
          </CardContent>
        </Card>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
          <GameBoardHeader
            connectionStatus="connected"
            phase={phase}
            roomCode={room.code}
            connectedPlayers={connectedPlayers}
            playerCount={players.length}
            isLobbyPhase={false}
            isMyTurn={Boolean(isMyTurn)}
            turnPlayerName={turnPlayerName}
            game={game}
          />

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
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
              onDiscardCard={(cardIndex) =>
                updateGame((current) => discardFromHand(current, cardIndex))
              }
              onDrawFromDeck={() => updateGame(drawFromDeck)}
              onDrawFromDiscard={() => updateGame(drawFromDiscard)}
              onPlayTable={(play) => updateGame((current) => applyTablePlay(current, play))}
              disableDraftSync
            />
          </div>
        </div>
      </div>
    </section>
  );
}

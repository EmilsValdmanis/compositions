import { useEffect, useState } from "react";
import { ClientOnly } from "@tanstack/react-router";
import { toast } from "sonner";
import { GameBoardView } from "#/components/game/game-board-view";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { GameResultsView } from "#/components/game/game-results-view";
import { playerName } from "#/components/game/game-view-helpers";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { GameRouteLoadingScreen } from "#/components/routes/game-route-loading-screen";

function initialRoomCode() {
  if (typeof window === "undefined") {
    return "";
  }

  return new URLSearchParams(window.location.search).get("room")?.toUpperCase() ?? "";
}

function roomShareUrl(code: string) {
  if (typeof window === "undefined") {
    return code;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("room", code);
  return url.toString();
}

export function ProtectedHome() {
  const {
    state,
    createRoom,
    joinRoom,
    leaveRoom,
    startGame,
    startNextRound,
    chooseDealing,
    drawFromDeck,
    drawFromDiscard,
    playTable,
    discardCard,
  } = useGameWebSocket();
  const [roomCode, setRoomCode] = useState(initialRoomCode);
  const players = state.room?.players ?? [];
  const currentPlayer = players.find((player) => player.playerId === state.playerId) ?? null;
  const connectedPlayers = players.filter((player) => player.connected).length;
  const phase = state.room?.phase ?? "lobby";
  const isLobbyPhase = !state.room || phase === "lobby";
  const isHost = currentPlayer?.isHost ?? false;
  const pendingDealChoice = state.room?.pendingDealChoice ?? null;
  const dealChooser =
    players.find((player) => player.playerId === pendingDealChoice?.chooserPlayerId) ?? null;
  const isDealChooser = pendingDealChoice?.chooserPlayerId === state.playerId;
  const allPlayersConnected = players.length > 0 && players.every((player) => player.connected);
  const canCreateRoom = state.connectionStatus === "connected" && !state.room;
  const canJoinRoom =
    state.connectionStatus === "connected" && !state.room && roomCode.trim() !== "";
  const canLeaveRoom = Boolean(state.room) && phase === "lobby";
  const canStartGame =
    Boolean(state.room) &&
    isHost &&
    players.length >= 2 &&
    allPlayersConnected &&
    pendingDealChoice == null;
  const isMyTurn = state.game?.turn.playerId === state.playerId;
  const canDraw = Boolean(state.game) && isMyTurn && !state.game?.turn.hasDrawn;
  const canDrawDeck =
    canDraw && !state.game?.turn.mustUseDiscardDraw && (state.game?.drawPileCount ?? 0) > 0;
  const topDiscardCard = state.game?.discardPile[0] ?? null;
  const canDrawDiscard = canDraw && Boolean(topDiscardCard);
  const canDiscard = Boolean(state.game) && isMyTurn && Boolean(state.game?.turn.hasDrawn);
  const turnPlayerName = playerName(players, state.game?.turn.playerId);
  const roundResultsGame = phase === "round_over" ? state.game : null;
  const completedGame = state.completedGame;
  const isBootstrappingConnection =
    state.connectionStatus === "idle" ||
    (state.connectionStatus === "connecting" && state.room === null && state.game === null);

  useEffect(() => {
    if (state.room?.code) {
      setRoomCode(state.room.code);
    }
  }, [state.room?.code]);

  useEffect(() => {
    if (!state.lastError) {
      return;
    }

    toast.error(state.lastError);
  }, [state.lastError, state.lastErrorId]);

  async function copyText(text: string, successMessage: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(successMessage);
    } catch {
      toast.error("Could not copy to clipboard");
    }
  }

  async function copyRoomCode() {
    if (!state.room?.code) {
      return;
    }

    await copyText(state.room.code, "Room code copied");
  }

  async function copyRoomLink() {
    if (!state.room?.code) {
      return;
    }

    await copyText(roomShareUrl(state.room.code), "Room link copied");
  }

  async function shareRoom() {
    if (!state.room?.code) {
      return;
    }

    const url = roomShareUrl(state.room.code);

    if (!navigator.share) {
      await copyText(url, "Room link copied");
      return;
    }

    try {
      await navigator.share({ title: "Compositions", text: `Join room ${state.room.code}`, url });
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        return;
      }
      toast.error("Could not share room link");
    }
  }

  async function handleDiscardCard(cardIndex: number) {
    if (!canDiscard) {
      toast.error("Draw before discarding");
      throw new Error("draw before discarding");
    }

    return discardCard(cardIndex);
  }

  async function handlePlayTable(play: Parameters<typeof playTable>[0]) {
    return playTable(play);
  }

  if (isBootstrappingConnection) {
    return <GameRouteLoadingScreen />;
  }

  return (
    <ClientOnly fallback={<GameRouteLoadingScreen />}>
      <section className="mx-auto flex h-full min-h-0 w-full flex-1 flex-col gap-3 md:gap-4">
        {roundResultsGame ? (
          <div key="round-results" className="flex min-h-0 flex-1 overflow-auto">
            <GameResultsView
              room={state.room}
              game={roundResultsGame}
              players={players}
              playerId={state.playerId}
              onStartNextRound={startNextRound}
            />
          </div>
        ) : isLobbyPhase ? (
          <div key="lobby">
            <GameLobbyView
              room={state.room}
              game={state.game}
              completedGame={completedGame}
              players={players}
              roomCode={roomCode}
              playerId={state.playerId}
              sessionId={state.sessionId}
              roomActions={{ canCreateRoom, canJoinRoom, canLeaveRoom, canStartGame }}
              dealChoice={{
                pendingDealChoice,
                dealChooserName: dealChooser?.name ?? null,
                isDealChooser: Boolean(isDealChooser),
              }}
              onRoomCodeChange={setRoomCode}
              onCreateRoom={createRoom}
              onJoinRoom={joinRoom}
              onStartGame={startGame}
              onChooseDealing={chooseDealing}
              onLeaveRoom={leaveRoom}
              onCopyRoomCode={copyRoomCode}
              onCopyRoomLink={copyRoomLink}
              onShareRoom={shareRoom}
            />
          </div>
        ) : (
          <div key="game" className="flex min-h-0 flex-1 flex-col">
            <GameBoardView
              game={state.game}
              roomCode={state.room?.code ?? null}
              playerId={state.playerId}
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
              onDrawFromDeck={drawFromDeck}
              onDrawFromDiscard={drawFromDiscard}
              onPlayTable={handlePlayTable}
            />
          </div>
        )}
      </section>
    </ClientOnly>
  );
}

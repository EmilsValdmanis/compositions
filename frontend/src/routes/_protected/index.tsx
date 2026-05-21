import { useEffect, useState } from "react";
import { type DragEndEvent } from "@dnd-kit/core";
import { ClientOnly, createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";
import { GameBoardHeader } from "#/components/game/game-board-header";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { GameBoardView } from "#/components/game/game-board-view";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { playerName } from "#/components/game/game-view-utils";
import { Spinner } from "#/components/ui/spinner";

export const Route = createFileRoute("/_protected/")({
  component: Home,
});

function GameRouteLoadingScreen() {
  return (
    <section className="flex flex-1 items-center justify-center">
      <div className="flex flex-col items-center gap-3 text-center">
        <Spinner className="size-8" />
        <div className="space-y-1">
          <p className="text-sm font-medium">Reconnecting to your game</p>
          <p className="text-muted-foreground text-sm">Loading the latest room state…</p>
        </div>
      </div>
    </section>
  );
}

function roomShareUrl(code: string) {
  if (typeof window === "undefined") {
    return code;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("room", code);
  return url.toString();
}

function Home() {
  return (
    <ClientOnly fallback={<GameRouteLoadingScreen />}>
      <HydratedHome />
    </ClientOnly>
  );
}

function HydratedHome() {
  const {
    state,
    createRoom,
    joinRoom,
    leaveRoom,
    startGame,
    chooseDealing,
    drawFromDeck,
    drawFromDiscard,
    playTable,
    discardCard,
  } = useGameWebSocket();
  const [roomCode, setRoomCode] = useState("");
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
  const isBootstrappingConnection =
    state.connectionStatus === "idle" ||
    (state.connectionStatus === "connecting" && state.room === null && state.game === null);

  useEffect(() => {
    const urlRoomCode = new URLSearchParams(window.location.search).get("room");
    if (urlRoomCode) {
      setRoomCode(urlRoomCode.toUpperCase());
    }
  }, []);

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

  function handleDragEnd(event: DragEndEvent) {
    if (event.over?.id !== "discard-pile") {
      return;
    }

    if (!canDiscard) {
      toast.error("Draw before discarding");
      return;
    }

    const cardIndex = event.active.data.current?.cardIndex;
    if (typeof cardIndex === "number") {
      discardCard(cardIndex);
    }
  }

  if (isBootstrappingConnection) {
    return <GameRouteLoadingScreen />;
  }

  return (
    <section className="mx-auto flex h-full min-h-0 w-full flex-1 flex-col gap-4 overflow-hidden">
      <GameBoardHeader
        connectionStatus={state.connectionStatus}
        phase={phase}
        roomCode={state.room?.code}
        connectedPlayers={connectedPlayers}
        playerCount={players.length}
        isLobbyPhase={isLobbyPhase}
        isMyTurn={Boolean(isMyTurn)}
        turnPlayerName={turnPlayerName}
        game={state.game}
      />

      {isLobbyPhase ? (
        <div key="lobby">
          <GameLobbyView
            room={state.room}
            game={state.game}
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
        <div key="game" className="flex min-h-0 flex-1 flex-col overflow-hidden">
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
            onDragEnd={handleDragEnd}
            onDrawFromDeck={drawFromDeck}
            onDrawFromDiscard={drawFromDiscard}
            onPlayTable={playTable}
          />
        </div>
      )}
    </section>
  );
}

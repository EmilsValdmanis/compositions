import { useEffect, useState } from "react";
import { type DragEndEvent } from "@dnd-kit/core";
import { createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";

import { useGameWebSocket } from "#/components/game-websocket-provider";
import { GameBoardView } from "#/components/game/game-board-view";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { formatLabel } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";

export const Route = createFileRoute("/_protected/")({
  component: Home,
});

function roomShareUrl(code: string) {
  if (typeof window === "undefined") {
    return code;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("room", code);
  return url.toString();
}

function Home() {
  const {
    state,
    connect,
    disconnect,
    createRoom,
    joinRoom,
    leaveRoom,
    startGame,
    chooseDealing,
    drawFromDeck,
    drawFromDiscard,
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
  const debugValue = {
    connectionStatus: state.connectionStatus,
    sessionId: state.sessionId,
    playerId: state.playerId,
    room: state.room,
    game: state.game,
    lastActionResult: state.lastActionResult,
    lastError: state.lastError,
    lastEvent: state.lastEvent,
  };

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

  return (
    <section className="mx-auto grid w-full gap-4">
      <Card size="sm" className="shadow-sm">
        <CardContent className="flex flex-col gap-3 py-0 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-2">
            <Badge variant={state.connectionStatus === "connected" ? "default" : "outline"}>
              {formatLabel(state.connectionStatus)}
            </Badge>
            <Badge variant="secondary">{formatLabel(phase)}</Badge>
            <Badge variant="outline">Room {state.room?.code ?? "None"}</Badge>
            <Badge variant="outline">
              {connectedPlayers}/{players.length || 0} online
            </Badge>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              onClick={() => void connect()}
              disabled={state.connectionStatus === "connecting"}
            >
              {state.sessionId ? "Reconnect" : "Connect"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={disconnect}
              disabled={
                state.connectionStatus === "idle" || state.connectionStatus === "disconnected"
              }
            >
              Disconnect
            </Button>
          </div>
        </CardContent>
      </Card>

      {isLobbyPhase ? (
        <div key="lobby">
          <GameLobbyView
            room={state.room}
            game={state.game}
            players={players}
            roomCode={roomCode}
            playerId={state.playerId}
            sessionId={state.sessionId}
            canCreateRoom={canCreateRoom}
            canJoinRoom={canJoinRoom}
            canLeaveRoom={canLeaveRoom}
            canStartGame={canStartGame}
            pendingDealChoice={pendingDealChoice}
            dealChooserName={dealChooser?.name ?? null}
            isDealChooser={Boolean(isDealChooser)}
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
        <div key="game">
          <GameBoardView
            debugValue={debugValue}
            game={state.game}
            phase={phase}
            players={players}
            connectedPlayers={connectedPlayers}
            canDrawDeck={canDrawDeck}
            canDrawDiscard={canDrawDiscard}
            canDiscard={canDiscard}
            isMyTurn={isMyTurn}
            topDiscardCard={topDiscardCard}
            onDragEnd={handleDragEnd}
            onDrawFromDeck={drawFromDeck}
            onDrawFromDiscard={drawFromDiscard}
          />
        </div>
      )}
    </section>
  );
}

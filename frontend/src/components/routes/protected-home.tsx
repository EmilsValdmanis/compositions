import { useEffect, useRef, useState } from "react";
import { ClientOnly, getRouteApi } from "@tanstack/react-router";
import { toast } from "sonner";
import { GameBoardView } from "#/components/game/game-board-view";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { GameResultsView } from "#/components/game/game-results-view";
import { playerName } from "#/components/game/game-view-helpers";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { GameRouteLoadingScreen } from "#/components/routes/game-route-loading-screen";

const protectedHomeRoute = getRouteApi("/_protected/");

function roomShareUrl(code: string) {
  if (typeof window === "undefined") {
    return code;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("room", code);
  return url.toString();
}

export function ProtectedHome() {
  const search = protectedHomeRoute.useSearch();
  const navigate = protectedHomeRoute.useNavigate();
  const {
    state,
    createRoom,
    joinRoom,
    leaveRoom,
    startGame,
    startNextRound,
    chooseDealing,
    sendEmote,
    drawFromDeck,
    drawFromDiscard,
    playTable,
    discardCard,
  } = useGameWebSocket();
  const autoJoinAttemptedRoomCodeRef = useRef<string | null>(null);
  const [roomCode, setRoomCode] = useState(search.room ?? "");
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
    if (
      state.connectionStatus !== "connected" ||
      state.room ||
      !search.room ||
      autoJoinAttemptedRoomCodeRef.current === search.room
    ) {
      return;
    }

    const inviteRoomCode = search.room;
    autoJoinAttemptedRoomCodeRef.current = inviteRoomCode;
    setRoomCode("");
    joinRoom(inviteRoomCode);
    void navigate({
      search: {},
      replace: true,
    });
  }, [joinRoom, navigate, search.room, state.connectionStatus, state.room]);

  useEffect(() => {
    if (search.room && state.room) {
      void navigate({
        search: {},
        replace: true,
      });
    }
  }, [navigate, search.room, state.room]);

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

  function handleRoomCodeChange(nextRoomCode: string) {
    const normalizedRoomCode = nextRoomCode.trim().toUpperCase();

    if (normalizedRoomCode !== autoJoinAttemptedRoomCodeRef.current) {
      autoJoinAttemptedRoomCodeRef.current = null;
    }

    setRoomCode(normalizedRoomCode);
  }

  function handleLeaveRoom() {
    autoJoinAttemptedRoomCodeRef.current = null;
    setRoomCode("");
    leaveRoom();
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
              dealChoice={{
                pendingDealChoice,
                dealChooserName: dealChooser?.name ?? null,
                isDealChooser: Boolean(isDealChooser),
              }}
              onStartNextRound={startNextRound}
              onChooseDealing={chooseDealing}
            />
          </div>
        ) : isLobbyPhase ? (
          <div key="lobby" className="flex min-h-0 flex-1 overflow-auto">
            <GameLobbyView
              room={state.room}
              game={state.game}
              completedGame={completedGame}
              players={players}
              roomCode={roomCode}
              roomActions={{ canCreateRoom, canJoinRoom, canLeaveRoom, canStartGame }}
              dealChoice={{
                pendingDealChoice,
                dealChooserName: dealChooser?.name ?? null,
                isDealChooser: Boolean(isDealChooser),
              }}
              onRoomCodeChange={handleRoomCodeChange}
              onCreateRoom={createRoom}
              onJoinRoom={joinRoom}
              onStartGame={startGame}
              onChooseDealing={chooseDealing}
              onLeaveRoom={handleLeaveRoom}
              onSendEmote={sendEmote}
              onCopyRoomCode={copyRoomCode}
              onCopyRoomLink={copyRoomLink}
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
              onSendEmote={sendEmote}
            />
          </div>
        )}
      </section>
    </ClientOnly>
  );
}

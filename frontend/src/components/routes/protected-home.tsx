import { copyTextToClipboard } from "#/lib/clipboard";
import { useEffect, useRef, useState } from "react";
import { ViewIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { ClientOnly, getRouteApi } from "@tanstack/react-router";
import { toast } from "sonner";
import { GameBoardView } from "#/components/game/game-board-view";
import { inferCompletedCompositionCollection } from "#/components/game/card-transfer-state";
import { EndGameProposalAlert } from "#/components/game/end-game-proposal-alert";
import { GameLobbyView } from "#/components/game/game-lobby-view";
import { GameResultsView } from "#/components/game/game-results-view";
import { playerName, playersForResults } from "#/components/game/game-view-helpers";
import { isFinishedGame, type GameMode, type GameSnapshot } from "#/lib/game-protocol";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { GameRouteLoadingScreen } from "#/components/routes/game-route-loading-screen";
import { Alert, AlertAction, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Button } from "#/components/ui/button";
import { isGameRouteSnapshotResolving } from "#/components/routes/game-route-view-state";
import { useGameSoundEvents } from "#/lib/game-sound-events";
import { playGameSound } from "#/lib/game-sounds";
import { pageTitle } from "#/lib/page-title";
import { isCompleteRoomCode } from "#/lib/room-code";
import { m } from "#/paraglide/messages.js";

async function copyText(text: string, successMessage: string) {
  try {
    await copyTextToClipboard(text);
    toast.success(successMessage);
  } catch {
    toast.error(m.clipboard_copy_failed());
  }
}

const protectedHomeRoute = getRouteApi("/_protected/");

function roomShareUrl(code: string) {
  if (typeof window === "undefined") {
    return code;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("room", code);
  return url.toString();
}

type RoundResultsTransition = {
  currentGame: GameSnapshot | null;
  isSpectating: boolean;
  waitingForCollection: { game: GameSnapshot; isSpectating: boolean } | null;
};

function useRoundResultsTransition(
  resultGame: GameSnapshot | null,
  currentGame: GameSnapshot | null,
  isSpectating: boolean,
) {
  const [transition, setTransition] = useState<RoundResultsTransition>({
    currentGame,
    isSpectating,
    waitingForCollection: null,
  });

  if (
    transition.currentGame !== currentGame ||
    transition.isSpectating !== isSpectating ||
    (!resultGame && transition.waitingForCollection)
  ) {
    let waitingForCollection = resultGame ? transition.waitingForCollection : null;
    if (
      !waitingForCollection &&
      resultGame === currentGame &&
      currentGame &&
      transition.currentGame &&
      inferCompletedCompositionCollection(transition.currentGame, currentGame)
    ) {
      waitingForCollection = { game: currentGame, isSpectating: transition.isSpectating };
    }

    setTransition({ currentGame, isSpectating, waitingForCollection });
  }

  function settleCollection(game: GameSnapshot) {
    setTransition((current) =>
      current.waitingForCollection?.game === game
        ? { ...current, waitingForCollection: null }
        : current,
    );
  }

  return { collection: transition.waitingForCollection, settleCollection };
}

function SpectatorNotice({ onStopSpectating }: { onStopSpectating: () => Promise<unknown> }) {
  return (
    <Alert className="pr-36">
      <HugeiconsIcon icon={ViewIcon} />
      <AlertTitle>{m.watching_game()}</AlertTitle>
      <AlertDescription>{m.spectator_hand_hidden()}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void onStopSpectating().catch(() => toast.error(m.social_action_failed()))}
        >
          {m.stop_watching()}
        </Button>
      </AlertAction>
    </Alert>
  );
}

export function ProtectedHome() {
  const search = protectedHomeRoute.useSearch();
  const navigate = protectedHomeRoute.useNavigate();
  const {
    state,
    createRoom,
    dismissError,
    joinRoom,
    leaveRoom,
    startGame,
    startNextRound,
    dismissCompletedGame,
    chooseDealing,
    sendEmote,
    sendFriendRequest,
    stopSpectating,
    drawFromDeck,
    drawFromDiscard,
    playTable,
    playTableAndDiscard,
    discardCard,
  } = useGameWebSocket();
  const autoJoinAttemptedRoomCodeRef = useRef<string | null>(null);
  const [roomCode, setRoomCode] = useState(search.room ?? "");
  const [gameMode, setGameMode] = useState<GameMode>("full");
  const players = state.room?.players ?? [];
  const activePlayers = players.filter((player) => !player.forfeited);
  const currentPlayer = players.find((player) => player.playerId === state.playerId) ?? null;
  const phase = state.room?.phase ?? "lobby";
  const isLobbyPhase = !state.room || phase === "lobby";
  const isGameInProgress = phase === "in_progress";
  const isHost = currentPlayer?.isHost ?? false;
  const pendingDealChoice = state.room?.pendingDealChoice ?? null;
  const dealChooser =
    players.find((player) => player.playerId === pendingDealChoice?.chooserPlayerId) ?? null;
  const dealChoice = {
    pendingDealChoice,
    dealChooserName: dealChooser?.name ?? null,
    isDealChooser: pendingDealChoice?.chooserPlayerId === state.playerId,
  };
  const allPlayersConnected =
    activePlayers.length > 0 && activePlayers.every((player) => player.connected);
  const canCreateRoom = state.connectionStatus === "connected" && !state.room;
  const canJoinRoom =
    state.connectionStatus === "connected" && !state.room && isCompleteRoomCode(roomCode);
  const canLeaveRoom = Boolean(state.room) && phase === "lobby";
  const canStartGame =
    Boolean(state.room) &&
    isHost &&
    players.length >= 2 &&
    allPlayersConnected &&
    pendingDealChoice == null;
  const isMyTurn = isGameInProgress && state.game?.turn.playerId === state.playerId;
  const canDraw = isGameInProgress && Boolean(state.game) && isMyTurn && !state.game?.turn.hasDrawn;
  const canDrawDeck =
    canDraw && !state.game?.turn.mustUseDiscardDraw && (state.game?.drawPileCount ?? 0) > 0;
  const topDiscardCard = state.game?.discardPile[0] ?? null;
  const canDrawDiscard = canDraw && Boolean(topDiscardCard);
  const canDiscard = Boolean(state.game) && isMyTurn && Boolean(state.game?.turn.hasDrawn);
  const turnPlayerName = playerName(players, state.game?.turn.playerId);
  const completedGame = state.completedGame;
  const isResultsPhase = phase === "round_over" || phase === "game_over";
  const currentResults =
    isResultsPhase && state.room && state.game && isFinishedGame(state.game)
      ? { room: state.room, game: state.game }
      : null;
  const results =
    currentResults ?? (completedGame && isFinishedGame(completedGame.game) ? completedGame : null);
  const boardGame = state.game ?? results?.game ?? null;
  const { collection, settleCollection } = useRoundResultsTransition(
    results?.game ?? null,
    boardGame,
    state.isSpectating,
  );
  const visibleRoundResults = collection ? null : results;
  const boardRoom = collection ? results?.room : state.room;
  const boardIsSpectating = collection?.isSpectating ?? state.isSpectating;
  const roundResultPlayers = playersForResults(visibleRoundResults?.room ?? null, state.room);
  const isBootstrappingConnection = isGameRouteSnapshotResolving(state);
  const currentPageTitle = visibleRoundResults
    ? m.results()
    : isLobbyPhase && !collection
      ? state.room
        ? m.lobby()
        : m.start()
      : m.board();

  useGameSoundEvents(state);

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
    dismissError();
  }, [dismissError, state.lastError, state.lastErrorId]);

  async function copyRoomCode() {
    if (!state.room?.code) {
      return;
    }

    await copyText(state.room.code, m.room_code_copied());
  }

  async function copyRoomLink() {
    if (!state.room?.code) {
      return;
    }

    await copyText(roomShareUrl(state.room.code), m.room_link_copied());
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

  async function handleDiscardCard(cardIndex: number, card: Parameters<typeof discardCard>[1]) {
    if (!canDiscard) {
      toast.error(m.draw_before_discarding());
      playGameSound("invalid-action");
      throw new Error("draw before discarding");
    }

    return discardCard(cardIndex, card);
  }

  if (isBootstrappingConnection) {
    return (
      <>
        <title>{pageTitle(m.loading())}</title>
        <GameRouteLoadingScreen />
      </>
    );
  }

  return (
    <ClientOnly fallback={<GameRouteLoadingScreen />}>
      <section className="mx-auto flex h-full min-h-0 w-full flex-1 flex-col gap-3 md:gap-4">
        <title>{pageTitle(currentPageTitle)}</title>
        <EndGameProposalAlert />
        {state.isSpectating ? <SpectatorNotice onStopSpectating={stopSpectating} /> : null}
        {visibleRoundResults ? (
          <div key="round-results" className="flex min-h-0 flex-1 overflow-auto">
            <GameResultsView
              room={visibleRoundResults.room}
              game={visibleRoundResults.game}
              players={roundResultPlayers}
              playerId={state.playerId}
              dealChoice={dealChoice}
              onStartNextRound={startNextRound}
              onBackToLobby={dismissCompletedGame}
              onChooseDealing={chooseDealing}
              onSendEmote={sendEmote}
              social={state.social}
              onSendFriendRequest={sendFriendRequest}
            />
          </div>
        ) : isLobbyPhase && !collection ? (
          <div key="lobby" className="flex min-h-0 flex-1 overflow-auto">
            <GameLobbyView
              room={state.room}
              game={state.game}
              completedGame={completedGame}
              players={players}
              roomCode={roomCode}
              roomActions={{
                canCreateRoom,
                canJoinRoom,
                canLeaveRoom,
                canStartGame,
                canSelectGameMode: isHost && pendingDealChoice == null,
              }}
              dealChoice={dealChoice}
              onRoomCodeChange={handleRoomCodeChange}
              onCreateRoom={createRoom}
              onJoinRoom={joinRoom}
              gameMode={
                state.room?.pendingDealChoice ? (state.room.gameMode ?? gameMode) : gameMode
              }
              onGameModeChange={setGameMode}
              onStartGame={() => startGame(gameMode)}
              onChooseDealing={chooseDealing}
              onLeaveRoom={handleLeaveRoom}
              onSendEmote={sendEmote}
              onCopyRoomCode={copyRoomCode}
              onCopyRoomLink={copyRoomLink}
              social={state.social}
              currentPlayerId={state.playerId}
              onSendFriendRequest={sendFriendRequest}
            />
          </div>
        ) : (
          <div key="game" className="flex min-h-0 flex-1 flex-col">
            <GameBoardView
              game={boardGame}
              roomCode={boardRoom?.code ?? null}
              playerId={state.playerId}
              players={boardRoom?.players ?? []}
              spectatorCount={boardRoom?.spectatorCount ?? 0}
              viewerMode={boardIsSpectating ? "spectator" : "player"}
              turnState={{
                canDrawDeck,
                canDrawDiscard,
                canDiscard,
                isMyTurn,
                turnPlayerName,
              }}
              topDiscardCard={boardGame?.discardPile[0] ?? null}
              onDiscardCard={handleDiscardCard}
              onDrawFromDeck={drawFromDeck}
              onDrawFromDiscard={drawFromDiscard}
              onPlayTable={playTable}
              onCollectionSettled={settleCollection}
              onPlayTableAndDiscard={playTableAndDiscard}
              onSendEmote={sendEmote}
              social={state.social}
              onSendFriendRequest={sendFriendRequest}
            />
          </div>
        )}
      </section>
    </ClientOnly>
  );
}

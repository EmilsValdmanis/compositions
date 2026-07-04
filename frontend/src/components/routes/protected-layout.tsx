import { Link, Outlet } from "@tanstack/react-router";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { Button } from "#/components/ui/button";
import { UserDropdown } from "#/components/user-dropdown";

function ProtectedLayoutStatus() {
  const { state } = useGameWebSocket();
  const roomCode = state.room?.code;
  const isLobbyPhase = !state.room || state.room.phase === "lobby";
  const turnPlayerId = state.game?.turn.playerId;
  const turnPlayerName = state.room?.players.find(
    (player) => player.playerId === turnPlayerId,
  )?.name;

  if (!roomCode) {
    return (
      <div className="min-w-0 text-center">
        <p className="truncate text-sm font-semibold tracking-[0.16em] uppercase text-foreground/90 md:text-base">
          Compositions
        </p>
      </div>
    );
  }

  const detail = isLobbyPhase
    ? "Waiting in lobby"
    : state.game
      ? `${turnPlayerName ? `${turnPlayerName}'s turn` : "In game"} · Round ${state.game.round} Turn ${state.game.turn.number}`
      : "In game";

  return (
    <div className="min-w-0 text-center">
      <p className="truncate text-sm font-semibold tracking-[0.16em] uppercase text-foreground/90 md:text-base">
        Compositions
        <span className="hidden font-normal normal-case tracking-normal text-muted-foreground sm:inline">
          {` · Room ${roomCode}`}
        </span>
      </p>
      <p className="truncate text-[0.72rem] text-muted-foreground">{detail}</p>
    </div>
  );
}

export function ProtectedLayout() {
  return (
    <>
      <nav className="w-full border-b bg-background/80 backdrop-blur-sm">
        <div className="grid w-full grid-cols-[auto_1fr_auto] items-center gap-3 px-4 py-2 md:px-6">
          <div className="justify-self-start">
            <ServerStatusBadge />
          </div>

          <ProtectedLayoutStatus />

          <div className="flex items-center justify-end gap-1 justify-self-end">
            {import.meta.env.DEV ? (
              <Button
                render={<Link to="/dev-ui" />}
                variant="outline"
                size="sm"
                className="hidden md:inline-flex"
              >
                Dev UI
              </Button>
            ) : null}
            <UserDropdown />
          </div>
        </div>
      </nav>
      <main className="flex min-h-0 w-full flex-1 flex-col gap-3 p-4 md:gap-4 md:p-6">
        <Outlet />
      </main>
    </>
  );
}

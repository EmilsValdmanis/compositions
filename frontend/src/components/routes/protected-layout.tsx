import { RankingIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, Outlet, getRouteApi } from "@tanstack/react-router";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { NotificationsDropdown } from "#/components/social/notifications-dropdown";
import { Button } from "#/components/ui/button";
import { UserDropdown } from "#/components/user-dropdown";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function ProtectedLayout() {
  return (
    <>
      <AppNavigation />
      <main className="flex min-h-0 w-full flex-1 flex-col gap-2 p-2 md:gap-4 md:p-6 [@media(max-height:600px)]:gap-2 [@media(max-height:600px)]:p-2">
        <Outlet />
      </main>
    </>
  );
}

export function AppNavigation() {
  const { session } = rootRouteApi.useRouteContext();

  return (
    <nav className="w-full border-b bg-background/80 backdrop-blur-sm">
      <div className="flex w-full items-center justify-between gap-2 px-2 py-2 sm:px-4 md:px-6">
        <Link
          to="/"
          aria-label={m.back_to_lobby()}
          className="inline-flex shrink-0 rounded-sm transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <img src="/favicon.svg" alt="" className="size-7" aria-hidden="true" />
        </Link>

        <div className="flex min-w-0 items-center justify-end gap-1">
          <ServerStatusBadge />

          <Button
            render={<Link to="/leaderboard" />}
            nativeButton={false}
            variant="outline"
            size="icon"
            aria-label={m.view_leaderboard()}
          >
            <HugeiconsIcon icon={RankingIcon} />
          </Button>

          <GameControlsMenu />
          {session ? <NotificationsDropdown /> : null}
          {session ? <UserDropdown /> : null}
        </div>
      </div>
    </nav>
  );
}

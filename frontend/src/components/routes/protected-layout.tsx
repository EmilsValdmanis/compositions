import { RankingIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, Outlet, getRouteApi } from "@tanstack/react-router";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { H6 } from "#/components/typography";
import { Button } from "#/components/ui/button";
import { UserDropdown } from "#/components/user-dropdown";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

function BrandTitle() {
  return (
    <span className="inline-flex min-w-0 items-center justify-center gap-2">
      <img src="/favicon.svg" alt="" className="size-5 shrink-0" aria-hidden="true" />
      <span className="truncate">{m.app_name()}</span>
    </span>
  );
}

function ProtectedLayoutTitle() {
  return (
    <Link
      to="/"
      aria-label={m.back_to_lobby()}
      className="min-w-0 rounded-md text-center transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <H6 className="truncate text-sm/5 tracking-[0.16em] uppercase text-foreground/90 md:text-base/6">
        <BrandTitle />
      </H6>
    </Link>
  );
}

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
      <div className="grid w-full grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3 px-4 py-2 md:px-6">
        <div className="min-w-0 justify-self-start">
          <ServerStatusBadge />
        </div>

        <ProtectedLayoutTitle />

        <div className="flex min-w-0 items-center justify-end gap-1 justify-self-end">
          <Button
            render={<Link to="/leaderboard" />}
            nativeButton={false}
            variant="ghost"
            size="icon"
            aria-label={m.view_leaderboard()}
          >
            <HugeiconsIcon icon={RankingIcon} />
          </Button>
          <GameControlsMenu />
          {session ? <UserDropdown /> : null}
        </div>
      </div>
    </nav>
  );
}

import { Outlet, getRouteApi } from "@tanstack/react-router";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { Text } from "#/components/typography";
import { UserDropdown } from "#/components/user-dropdown";

const rootRouteApi = getRouteApi("__root__");

function BrandTitle() {
  return (
    <span className="inline-flex min-w-0 items-center justify-center gap-2">
      <img src="/favicon.svg" alt="" className="size-5 shrink-0" aria-hidden="true" />
      <span className="truncate">Compositions</span>
    </span>
  );
}

function ProtectedLayoutTitle() {
  return (
    <div className="min-w-0 text-center">
      <Text as="p" variant="brand" className="truncate">
        <BrandTitle />
      </Text>
    </div>
  );
}

export function ProtectedLayout() {
  return (
    <>
      <AppNavigation />
      <main className="flex min-h-0 w-full flex-1 flex-col gap-3 p-4 md:gap-4 md:p-6">
        <Outlet />
      </main>
    </>
  );
}

export function AppNavigation() {
  const { session } = rootRouteApi.useRouteContext();

  return (
    <nav className="w-full border-b bg-background/80 backdrop-blur-sm">
      <div className="grid w-full grid-cols-[auto_1fr_auto] items-center gap-3 px-4 py-2 md:px-6">
        <div className="justify-self-start">
          <ServerStatusBadge />
        </div>

        <ProtectedLayoutTitle />

        <div className="flex items-center justify-end gap-1 justify-self-end">
          <GameControlsMenu />
          {session ? <UserDropdown /> : null}
        </div>
      </div>
    </nav>
  );
}

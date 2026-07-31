import { Outlet, getRouteApi } from "@tanstack/react-router";
import { AppSidebar } from "#/components/app-sidebar";
import { GlobalConnectionRecoveryDialog } from "#/components/connection-recovery-dialog";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { NotificationsDropdown } from "#/components/social/notifications-dropdown";
import { ThemeSwitcher } from "#/components/theme-switcher";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "#/components/ui/sidebar";

const rootRouteApi = getRouteApi("__root__");

export function ProtectedLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}

function AppShell({ children }: { children: React.ReactNode }) {
  const { session, sidebarOpen } = rootRouteApi.useRouteContext();

  return (
    <SidebarProvider
      defaultOpen={sidebarOpen === "true"}
      className="h-full min-h-0! flex-1"
      style={{ "--sidebar-width": "18rem" } as React.CSSProperties}
    >
      <AppSidebar />
      <SidebarInset className="min-h-0 overflow-hidden">
        <AppHeader />
        <div className="flex min-h-0 w-full flex-1 flex-col p-2 gap-2 md:gap-4 [@media(max-height:600px)]:gap-2 overflow-y-auto">
          {children}
        </div>
      </SidebarInset>
      {session ? <GlobalConnectionRecoveryDialog /> : null}
    </SidebarProvider>
  );
}

function AppHeader() {
  const { session } = rootRouteApi.useRouteContext();

  return (
    <header className="flex shrink-0 items-center border-b md:border-none gap-2 h-12 md:h-16 px-2">
      <SidebarTrigger />
      <div className="ml-auto flex shrink-0 items-center  gap-1">
        <GameControlsMenu />
        {session ? <NotificationsDropdown /> : null}
        <ThemeSwitcher />
      </div>
    </header>
  );
}

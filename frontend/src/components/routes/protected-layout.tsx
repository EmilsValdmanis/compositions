import { Outlet, getRouteApi } from "@tanstack/react-router";
import { AppSidebar } from "#/components/app-sidebar";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { NotificationsDropdown } from "#/components/social/notifications-dropdown";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "#/components/ui/sidebar";

const rootRouteApi = getRouteApi("__root__");

export function ProtectedLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const { sidebarOpen } = rootRouteApi.useRouteContext();

  return (
    <SidebarProvider
      defaultOpen={sidebarOpen === "true"}
      className="h-full min-h-0! flex-1"
      style={{ "--sidebar-width": "18rem" } as React.CSSProperties}
    >
      <AppSidebar />
      <SidebarInset className="min-h-0 overflow-hidden">
        <AppHeader />
        <div className="flex min-h-0 w-full flex-1 flex-col p-2 md:pb-1 gap-2 md:gap-4 [@media(max-height:600px)]:gap-2 overflow-y-auto">
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

function AppHeader() {
  const { session } = rootRouteApi.useRouteContext();

  return (
    <header className="flex shrink-0 items-center gap-2 h-12 md:h-16 px-2">
      <SidebarTrigger />
      <div className="ml-auto flex shrink-0 items-center gap-1">
        <ServerStatusBadge />
        {session ? <NotificationsDropdown /> : null}
      </div>
    </header>
  );
}

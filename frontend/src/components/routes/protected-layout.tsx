import { Link, Outlet, getRouteApi } from "@tanstack/react-router";
import { AppSidebar } from "#/components/app-sidebar";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { NotificationsDropdown } from "#/components/social/notifications-dropdown";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "#/components/ui/breadcrumb";
import { Separator } from "#/components/ui/separator";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "#/components/ui/sidebar";
import { appPageLabel, useAppPage } from "#/lib/app-navigation";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function ProtectedLayout() {
  return (
    <AppShell contentClassName="gap-2 md:gap-4 [@media(max-height:600px)]:gap-2 [@media(max-height:600px)]:p-2">
      <Outlet />
    </AppShell>
  );
}

export function AppShell({
  children,
  contentClassName,
}: {
  children: React.ReactNode;
  contentClassName?: string;
}) {
  return (
    <SidebarProvider
      className="h-full min-h-0! flex-1"
      style={{ "--sidebar-width": "19rem" } as React.CSSProperties}
    >
      <AppSidebar />
      <SidebarInset className="min-h-0 overflow-hidden">
        <AppHeader />
        <div
          className={cn(
            "flex min-h-0 w-full flex-1 flex-col p-2 pt-0 md:p-6 md:pt-0",
            contentClassName,
          )}
        >
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

function AppHeader() {
  const { session } = rootRouteApi.useRouteContext();
  const currentPage = useAppPage();

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 px-3 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12 md:px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-1 data-[orientation=vertical]:h-4" />
      <Breadcrumb className="min-w-0">
        <BreadcrumbList className="flex-nowrap">
          <BreadcrumbItem className="hidden sm:inline-flex">
            <BreadcrumbLink render={<Link to="/" />}>{m.app_name()}</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator className="hidden sm:block" />
          <BreadcrumbItem className="min-w-0">
            <BreadcrumbPage className="truncate">{appPageLabel(currentPage)}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto flex shrink-0 items-center gap-1">
        <ServerStatusBadge />
        {session ? <NotificationsDropdown /> : null}
      </div>
    </header>
  );
}

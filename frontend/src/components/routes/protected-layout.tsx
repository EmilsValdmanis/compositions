import { Outlet } from "@tanstack/react-router";
import { AppSidebar } from "#/components/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "#/components/ui/sidebar";
import { cn } from "#/lib/utils";

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
    <SidebarProvider className="h-full min-h-0! flex-1">
      <AppSidebar />
      <SidebarInset className="min-h-0 overflow-hidden">
        <SidebarTrigger className="absolute top-2 left-2 md:hidden" />
        <div
          className={cn("flex min-h-0 w-full flex-1 flex-col p-2 pt-12 md:p-6", contentClassName)}
        >
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

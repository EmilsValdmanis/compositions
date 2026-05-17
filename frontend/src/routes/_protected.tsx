import { ModeToggle } from "#/components/mode-toggle";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { UserDropdown } from "#/components/user-dropdown";
import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_protected")({
  beforeLoad: ({ context }) => {
    if (!context.session) {
      throw redirect({ to: "/sign-in" });
    }
  },
  component: ProtectedLayout,
});

function ProtectedLayout() {
  return (
    <>
      <div className="border-b w-full grid grid-cols-3 items-center py-2 px-4">
        <ServerStatusBadge />
        <h1 className="font-bold text-center">Compositions</h1>
        <div className="flex items-center justify-end gap-1">
          <UserDropdown />
          <ModeToggle />
        </div>
      </div>
      <Outlet />
    </>
  );
}

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
      <nav className="w-full border-b">
        <div className="grid w-full grid-cols-3 items-center px-4 py-2">
          <ServerStatusBadge />
          <h1 className="text-center text-lg font-semibold tracking-tight md:text-xl">
            Compositions
          </h1>
          <div className="flex items-center justify-end gap-1">
            <UserDropdown />
          </div>
        </div>
      </nav>
      <main className="flex w-full flex-1 flex-col gap-4 p-4 md:p-6">
        <Outlet />
      </main>
    </>
  );
}

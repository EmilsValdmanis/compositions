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
      <nav className="w-full flex justify-center border-b">
        <div className="grow grid grid-cols-3 items-center py-2 px-4 max-w-7xl">
          <ServerStatusBadge />
          <h1 className="text-center text-lg font-semibold tracking-tight md:text-xl">
            Compositions
          </h1>
          <div className="flex items-center justify-end gap-1">
            <UserDropdown />
          </div>
        </div>
      </nav>
      <main className="flex flex-col gap-4 p-4 md:p-8">
        <Outlet />
      </main>
    </>
  );
}

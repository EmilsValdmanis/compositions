import { Link, Outlet } from "@tanstack/react-router";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { UserDropdown } from "#/components/user-dropdown";
import { Button } from "#/components/ui/button";

export function ProtectedLayout() {
  return (
    <>
      <nav className="w-full border-b">
        <div className="grid w-full grid-cols-3 items-center px-4 py-2">
          <ServerStatusBadge />
          <h1 className="text-center text-lg font-semibold tracking-tight md:text-xl">
            Compositions
          </h1>

          <div className="flex items-center justify-end gap-1">
            <button
              type="button"
              onClick={() => {
                throw new Error("Sentry Test Error");
              }}
            >
              Break the world
            </button>
            {import.meta.env.DEV ? (
              <Button asChild variant="outline" size="sm" className="hidden md:inline-flex">
                <Link to="/dev-ui">Dev UI</Link>
              </Button>
            ) : null}
            <UserDropdown />
          </div>
        </div>
      </nav>
      <main className="flex min-h-0 w-full flex-1 flex-col gap-4 p-4 md:p-6">
        <Outlet />
      </main>
    </>
  );
}

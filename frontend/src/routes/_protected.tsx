import { ModeToggle } from "#/components/mode-toggle";
import { ServerStatusBadge } from "#/components/server-status-badge";
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
      <Outlet />
      <ServerStatusBadge className="absolute top-4 left-4" />
      <ModeToggle className="absolute top-4 right-4" />
    </>
  );
}

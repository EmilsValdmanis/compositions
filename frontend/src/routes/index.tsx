import { GameWebSocketActions } from "#/components/game-websocket-actions";
import { ModeToggle } from "#/components/mode-toggle";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { createFileRoute } from "@tanstack/react-router";
import { getUserInitials } from "#/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import SignInButton from "#/components/auth/sign-in-button";
import SignOutButton from "#/components/auth/sign-out-button";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  const { session } = Route.useRouteContext();

  const user = session?.user;
  const displayName = user?.name || "";
  const initials = getUserInitials(displayName);

  return (
    <main className="h-screen flex items-center justify-center gap-4 flex-col">
      <div className="flex items-center gap-4">
        <h1>Compositions</h1>
        <ServerStatusBadge />
        <ModeToggle />
      </div>
      {!session && <SignInButton />}
      {session && (
        <div className="flex items-center gap-4">
          <Avatar>
            <AvatarImage src={user?.image || ""} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          <p className="text-xs">{displayName}</p>
          <SignOutButton />
        </div>
      )}
      <GameWebSocketActions />
    </main>
  );
}

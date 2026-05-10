import { authClient } from "#/lib/auth-client";
import { ModeToggle } from "#/components/mode-toggle";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { createFileRoute, useRouter } from "@tanstack/react-router";
import { getUserInitials } from "#/lib/utils";
import { Button } from "#/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { HugeiconsIcon } from "@hugeicons/react";
import { Login01Icon, Logout02FreeIcons } from "@hugeicons/core-free-icons";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  const { session } = Route.useRouteContext();
  const router = useRouter();

  const handleGoogleSignIn = async () => {
    await authClient.signIn.social({
      provider: "google",
    });
  };

  const handleSignOut = async () => {
    await authClient.signOut();
    await router.invalidate();
  };

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
      {!session && (
        <Button onClick={handleGoogleSignIn}>
          Sign in <HugeiconsIcon icon={Login01Icon} />
        </Button>
      )}
      {session && (
        <div className="flex items-center gap-4">
          <Avatar>
            <AvatarImage src={user?.image || ""} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          <p className="text-xs">{displayName}</p>
          <Button variant="destructive" size="icon" onClick={handleSignOut}>
            <HugeiconsIcon icon={Logout02FreeIcons} />
          </Button>
        </div>
      )}
    </main>
  );
}

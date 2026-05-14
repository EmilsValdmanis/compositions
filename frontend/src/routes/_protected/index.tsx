import { createFileRoute } from "@tanstack/react-router";
import { getUserInitials } from "#/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import SignOutButton from "#/components/auth/sign-out-button";

export const Route = createFileRoute("/_protected/")({
  component: Home,
});

function Home() {
  const { session } = Route.useRouteContext();

  const user = session?.user;
  const displayName = user?.name || "";
  const initials = getUserInitials(displayName);

  return (
    <>
      <div className="flex items-center gap-4">
        <h1>Compositions</h1>
      </div>
      <div className="flex items-center gap-4">
        <Avatar>
          <AvatarImage src={user?.image || ""} />
          <AvatarFallback>{initials}</AvatarFallback>
        </Avatar>
        <p className="text-xs">{displayName}</p>
        <SignOutButton />
      </div>
    </>
  );
}

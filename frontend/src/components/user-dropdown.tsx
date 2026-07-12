import { useState, useSyncExternalStore } from "react";
import {
  BookOpen01Icon,
  CodeXmlIcon,
  Logout02FreeIcons,
  VolumeHighIcon,
  VolumeOffIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, getRouteApi, useRouter } from "@tanstack/react-router";
import { GameRulesDialog } from "#/components/game/game-rules-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { useTheme } from "#/components/theme-provider";
import { authClient } from "#/lib/auth-client";
import {
  areGameSoundsEnabled,
  setGameSoundsEnabled,
  subscribeToGameSoundPreference,
} from "#/lib/game-sounds";
import { getUserInitials } from "#/lib/utils";

const rootRouteApi = getRouteApi("__root__");

export function UserDropdown() {
  const { session } = rootRouteApi.useRouteContext();
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const [rulesOpen, setRulesOpen] = useState(false);
  const soundsEnabled = useSyncExternalStore(
    subscribeToGameSoundPreference,
    areGameSoundsEnabled,
    () => true,
  );

  const user = session?.user;
  const displayName = user?.name || "";
  const initials = getUserInitials(displayName);

  const handleSignOut = async () => {
    await authClient.signOut();
    await router.invalidate();
  };

  function toggleSounds() {
    setGameSoundsEnabled(!soundsEnabled);
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full"
              aria-label="Open account menu"
            />
          }
        >
          <Avatar>
            <AvatarImage src={user?.image || ""} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuGroup>
            <DropdownMenuLabel className="flex flex-col gap-0.5">
              <span className="font-medium text-foreground">{displayName || "Account"}</span>
              {user?.email ? (
                <span className="text-muted-foreground text-xs">{user.email}</span>
              ) : null}
            </DropdownMenuLabel>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem onClick={() => setRulesOpen(true)}>
              <HugeiconsIcon icon={BookOpen01Icon} />
              Rules
            </DropdownMenuItem>
            <DropdownMenuItem onClick={toggleSounds}>
              <HugeiconsIcon icon={soundsEnabled ? VolumeHighIcon : VolumeOffIcon} />
              Sound effects
              <span className="ml-auto text-xs text-muted-foreground">
                {soundsEnabled ? "On" : "Off"}
              </span>
            </DropdownMenuItem>
            {import.meta.env.DEV ? (
              <DropdownMenuItem render={<Link to="/dev-ui" />}>
                <HugeiconsIcon icon={CodeXmlIcon} />
                Dev UI
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>Theme</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuGroup>
                  <DropdownMenuRadioGroup
                    value={theme}
                    onValueChange={(value) => setTheme(value as typeof theme)}
                  >
                    <DropdownMenuRadioItem value="light">Light</DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="dark">Dark</DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="system">System</DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem onClick={handleSignOut}>
              <HugeiconsIcon icon={Logout02FreeIcons} />
              Sign Out
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <GameRulesDialog open={rulesOpen} onOpenChange={setRulesOpen} />
    </>
  );
}

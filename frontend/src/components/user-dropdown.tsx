import { useState, useSyncExternalStore } from "react";
import {
  BookOpen01Icon,
  Logout02FreeIcons,
  Motion01Icon,
  UserIcon,
  VolumeHighIcon,
  VolumeOffIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, getRouteApi, useRouter } from "@tanstack/react-router";
import { toast } from "sonner";
import { GameRulesDialog } from "#/components/game/game-rules-dialog";
import { LanguageSubmenu } from "#/components/language-switcher";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { SidebarMenuButton, useSidebar } from "#/components/ui/sidebar";
import { Caption, P } from "#/components/typography";
import { authClient } from "#/lib/auth-client";
import {
  areGameSoundsEnabled,
  setGameSoundsEnabled,
  subscribeToGameSoundPreference,
} from "#/lib/game-sounds";
import {
  setReducedMotionPreferenceEnabled,
  useReducedMotionPreference,
} from "#/lib/reduced-motion";
import { getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function UserDropdown({ presentation = "button" }: { presentation?: "button" | "sidebar" }) {
  const { session } = rootRouteApi.useRouteContext();
  const { isMobile } = useSidebar();
  const router = useRouter();
  const [rulesOpen, setRulesOpen] = useState(false);
  const soundsEnabled = useSyncExternalStore(
    subscribeToGameSoundPreference,
    areGameSoundsEnabled,
    () => true,
  );
  const reduceMotion = useReducedMotionPreference();

  const user = session?.user;
  const displayName = user?.name || "";
  const initials = getUserInitials(displayName);

  const handleSignOut = async () => {
    try {
      await authClient.signOut();
      await router.invalidate();
    } catch {
      toast.error(m.error_sign_out());
    }
  };

  function toggleSounds() {
    setGameSoundsEnabled(!soundsEnabled);
  }

  function toggleReducedMotion() {
    setReducedMotionPreferenceEnabled(!reduceMotion);
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            presentation === "sidebar" ? (
              <SidebarMenuButton size="lg" tooltip={m.account_menu()} />
            ) : (
              <Button variant="ghost" size="icon" aria-label={m.account_menu()} />
            )
          }
        >
          <Avatar>
            <AvatarImage src={user?.image || ""} />
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          {presentation === "sidebar" ? (
            <span className="flex min-w-0 flex-col items-start">
              <span className="truncate font-medium">{displayName || m.account()}</span>
              {user?.email ? (
                <span className="truncate text-xs text-muted-foreground">{user.email}</span>
              ) : null}
            </span>
          ) : null}
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          side={presentation === "sidebar" && !isMobile ? "right" : "bottom"}
          sideOffset={4}
          className="w-56"
        >
          <DropdownMenuGroup>
            <DropdownMenuLabel className="flex flex-col gap-0.5">
              <P size="sm" className="font-medium text-foreground">
                {displayName || m.account()}
              </P>
              {user?.email ? <Caption>{user.email}</Caption> : null}
            </DropdownMenuLabel>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            {user?.id ? (
              <DropdownMenuItem
                render={<Link to="/players/$playerId" params={{ playerId: user.id }} />}
              >
                <HugeiconsIcon icon={UserIcon} />
                {m.profile()}
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuItem onClick={() => setRulesOpen(true)}>
              <HugeiconsIcon icon={BookOpen01Icon} />
              {m.rules()}
            </DropdownMenuItem>
            <DropdownMenuCheckboxItem
              checked={soundsEnabled}
              onCheckedChange={toggleSounds}
              showUncheckedIndicator
            >
              <HugeiconsIcon icon={soundsEnabled ? VolumeHighIcon : VolumeOffIcon} />
              {m.sound_effects()}
            </DropdownMenuCheckboxItem>
            <DropdownMenuCheckboxItem
              checked={reduceMotion}
              onCheckedChange={toggleReducedMotion}
              showUncheckedIndicator
            >
              <HugeiconsIcon icon={Motion01Icon} />
              {m.reduce_motion()}
            </DropdownMenuCheckboxItem>
            <LanguageSubmenu />
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem onClick={handleSignOut}>
              <HugeiconsIcon icon={Logout02FreeIcons} />
              {m.sign_out()}
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <GameRulesDialog open={rulesOpen} onOpenChange={setRulesOpen} />
    </>
  );
}

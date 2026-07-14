import { useState, useSyncExternalStore } from "react";
import {
  BookOpen01Icon,
  CodeXmlIcon,
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
import { Strong, Text } from "#/components/typography";
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
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full"
              aria-label={m.account_menu()}
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
              <Strong className="text-foreground">{displayName || m.account()}</Strong>
              {user?.email ? <Text variant="caption">{user.email}</Text> : null}
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
            <DropdownMenuItem onClick={toggleSounds}>
              <HugeiconsIcon icon={soundsEnabled ? VolumeHighIcon : VolumeOffIcon} />
              {m.sound_effects()}
              <Text variant="caption" className="ml-auto">
                {soundsEnabled ? m.on() : m.off()}
              </Text>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={toggleReducedMotion}>
              <HugeiconsIcon icon={Motion01Icon} />
              {m.reduce_motion()}
              <Text variant="caption" className="ml-auto">
                {reduceMotion ? m.on() : m.off()}
              </Text>
            </DropdownMenuItem>
            {import.meta.env.DEV ? (
              <DropdownMenuItem render={<Link to="/dev-ui" />}>
                <HugeiconsIcon icon={CodeXmlIcon} />
                {m.dev_ui()}
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>{m.theme()}</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuGroup>
                  <DropdownMenuRadioGroup
                    value={theme}
                    onValueChange={(value) => setTheme(value as typeof theme)}
                  >
                    <DropdownMenuRadioItem value="light">{m.theme_light()}</DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="dark">{m.theme_dark()}</DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="system">{m.theme_system()}</DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
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

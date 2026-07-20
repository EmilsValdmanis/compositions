import { Home01Icon, RankingIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, getRouteApi } from "@tanstack/react-router";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { SidebarFriendsList } from "#/components/social/friends-list";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  SidebarSeparator,
  useSidebar,
} from "#/components/ui/sidebar";
import { UserDropdown } from "#/components/user-dropdown";
import { useAppPage } from "#/lib/app-navigation";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function AppSidebar() {
  const { session } = rootRouteApi.useRouteContext();
  const { state, sendGameInvite } = useGameWebSocket();
  const { setOpenMobile } = useSidebar();
  const currentPage = useAppPage();
  const players = state.room?.players ?? [];
  const canInvite = Boolean(
    state.room &&
    state.room.phase === "lobby" &&
    !state.room.pendingDealChoice &&
    players.length < 4,
  );
  const unavailableUserIds = players.flatMap((player) => (player.userId ? [player.userId] : []));

  function closeMobileSidebar() {
    setOpenMobile(false);
  }

  return (
    <Sidebar side="left" variant="floating" collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              tooltip={m.back_to_lobby()}
              render={<Link to="/" onClick={closeMobileSidebar} />}
            >
              <img src="/favicon.svg" alt="" className="size-8" aria-hidden="true" />
              <span className="font-heading font-semibold">{m.app_name()}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent className="overflow-hidden">
        <SidebarGroup className="shrink-0">
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip={m.lobby()}
                  isActive={currentPage === "lobby"}
                  render={<Link to="/" onClick={closeMobileSidebar} />}
                >
                  <HugeiconsIcon icon={Home01Icon} />
                  <span>{m.lobby()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip={m.leaderboard()}
                  isActive={currentPage === "leaderboard"}
                  render={<Link to="/leaderboard" onClick={closeMobileSidebar} />}
                >
                  <HugeiconsIcon icon={RankingIcon} />
                  <span>{m.leaderboard()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <GameControlsMenu presentation="sidebar" />
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        {session ? (
          <>
            <SidebarSeparator />
            <SidebarFriendsList
              friends={state.social.friends}
              canInvite={canInvite}
              unavailableUserIds={unavailableUserIds}
              onInvite={sendGameInvite}
            />
          </>
        ) : null}
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          {session ? (
            <SidebarMenuItem>
              <UserDropdown presentation="sidebar" />
            </SidebarMenuItem>
          ) : null}
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

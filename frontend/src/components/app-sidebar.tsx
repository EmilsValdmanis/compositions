import {
  CodeXmlIcon,
  Home01Icon,
  JokerIcon,
  RankingIcon,
  ShieldUserIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, getRouteApi } from "@tanstack/react-router";
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
  useSidebar,
} from "#/components/ui/sidebar";
import { UserDropdown } from "#/components/user-dropdown";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function AppSidebar() {
  const { session } = rootRouteApi.useRouteContext();
  const { state, dismissCompletedGame, removeFriend, sendGameInvite, spectateGame } =
    useGameWebSocket();
  const { setOpenMobile } = useSidebar();
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

  function goToLobby() {
    dismissCompletedGame();
    closeMobileSidebar();
  }

  return (
    <Sidebar side="left" variant="floating" collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton className="pointer-events-none">
              <HugeiconsIcon icon={JokerIcon} className="text-primary" />
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
                  render={
                    <Link
                      to="/"
                      activeOptions={{ exact: true }}
                      activeProps={{ "data-active": true }}
                      onClick={goToLobby}
                    />
                  }
                >
                  <HugeiconsIcon icon={Home01Icon} />
                  <span>{m.lobby()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip={m.leaderboard()}
                  render={
                    <Link
                      to="/leaderboard"
                      activeOptions={{ exact: true }}
                      activeProps={{ "data-active": true }}
                      onClick={closeMobileSidebar}
                    />
                  }
                >
                  <HugeiconsIcon icon={RankingIcon} />
                  <span>{m.leaderboard()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              {session?.user.isAdmin ? (
                <SidebarMenuItem>
                  <SidebarMenuButton
                    tooltip={m.admin()}
                    render={
                      <Link
                        to="/admin"
                        activeOptions={{ exact: true }}
                        activeProps={{ "data-active": true }}
                        onClick={closeMobileSidebar}
                      />
                    }
                  >
                    <HugeiconsIcon icon={ShieldUserIcon} />
                    <span>{m.admin()}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : null}
              {import.meta.env.DEV ? (
                <SidebarMenuItem>
                  <SidebarMenuButton
                    tooltip={m.dev_ui()}
                    render={
                      <Link
                        to="/dev-ui"
                        activeOptions={{ exact: true }}
                        activeProps={{ "data-active": true }}
                        onClick={closeMobileSidebar}
                      />
                    }
                  >
                    <HugeiconsIcon icon={CodeXmlIcon} />
                    <span>{m.dev_ui()}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : null}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        {session ? (
          <>
            <SidebarFriendsList
              friends={state.social.friends}
              isLoading={state.connectionStatus !== "connected" || state.social.userId === ""}
              canInvite={canInvite}
              canSpectate={!state.room || state.isSpectating}
              unavailableUserIds={unavailableUserIds}
              onInvite={sendGameInvite}
              onSpectate={spectateGame}
              onUnfriend={removeFriend}
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
    </Sidebar>
  );
}

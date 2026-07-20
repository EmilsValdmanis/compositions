import { Home01Icon, RankingIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, getRouteApi, useMatchRoute } from "@tanstack/react-router";
import { GameControlsMenu } from "#/components/game/game-controls-menu";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { NotificationsDropdown } from "#/components/social/notifications-dropdown";
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
  useSidebar,
} from "#/components/ui/sidebar";
import { UserDropdown } from "#/components/user-dropdown";
import { m } from "#/paraglide/messages.js";

const rootRouteApi = getRouteApi("__root__");

export function AppSidebar() {
  const { session } = rootRouteApi.useRouteContext();
  const matchRoute = useMatchRoute();
  const { setOpenMobile } = useSidebar();
  const isLobby = Boolean(matchRoute({ to: "/", fuzzy: false }));
  const isLeaderboard = Boolean(matchRoute({ to: "/leaderboard", fuzzy: false }));

  function closeMobileSidebar() {
    setOpenMobile(false);
  }

  return (
    <Sidebar side="left" collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              tooltip={m.back_to_lobby()}
              isActive={isLobby}
              render={<Link to="/" onClick={closeMobileSidebar} />}
            >
              <img src="/favicon.svg" alt="" className="size-8" aria-hidden="true" />
              <span className="font-heading font-semibold">{m.app_name()}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip={m.lobby()}
                  isActive={isLobby}
                  render={<Link to="/" onClick={closeMobileSidebar} />}
                >
                  <HugeiconsIcon icon={Home01Icon} />
                  <span>{m.lobby()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip={m.leaderboard()}
                  isActive={isLeaderboard}
                  render={<Link to="/leaderboard" onClick={closeMobileSidebar} />}
                >
                  <HugeiconsIcon icon={RankingIcon} />
                  <span>{m.leaderboard()}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              {session ? (
                <SidebarMenuItem>
                  <NotificationsDropdown presentation="sidebar" />
                </SidebarMenuItem>
              ) : null}
              <SidebarMenuItem>
                <GameControlsMenu presentation="sidebar" />
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <ServerStatusBadge presentation="sidebar" />
          </SidebarMenuItem>
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

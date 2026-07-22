import {
  GameController03Icon,
  UserGroupIcon,
  UserIcon,
  UserRemove01Icon,
  ViewIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { type SocialUser } from "#/components/game-websocket-provider";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "#/components/ui/sidebar";
import { Skeleton } from "#/components/ui/skeleton";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

export function formatGameDuration(startedAt: string, now = Date.now()) {
  const startedAtMs = Date.parse(startedAt);
  const totalMinutes = Number.isFinite(startedAtMs)
    ? Math.max(0, Math.floor((now - startedAtMs) / 60_000))
    : 0;
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
}

export function SidebarFriendsList({
  friends,
  isLoading = false,
  canInvite,
  canSpectate = true,
  unavailableUserIds = [],
  onInvite,
  onSpectate,
  onUnfriend,
}: {
  friends: SocialUser[];
  isLoading?: boolean;
  canInvite: boolean;
  canSpectate?: boolean;
  unavailableUserIds?: string[];
  onInvite?: (userId: string) => Promise<unknown>;
  onSpectate?: (userId: string) => Promise<unknown>;
  onUnfriend?: (userId: string) => Promise<unknown>;
}) {
  const { isMobile, setOpenMobile } = useSidebar();
  const navigate = useNavigate();
  const [pendingUserIds, setPendingUserIds] = useState<Set<string>>(new Set());
  const [now, setNow] = useState(() => Date.now());
  const unavailable = new Set(unavailableUserIds);
  const sortedFriends = friends.toSorted(
    (left, right) =>
      Number(Boolean(right.activeGame)) - Number(Boolean(left.activeGame)) ||
      Number(right.online) - Number(left.online) ||
      left.name.localeCompare(right.name),
  );

  useEffect(() => {
    if (!friends.some((friend) => friend.activeGame)) {
      return;
    }
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, [friends]);

  async function invite(friend: SocialUser) {
    if (!onInvite) return;
    setPendingUserIds((current) => new Set(current).add(friend.id));
    try {
      await onInvite(friend.id);
      toast.success(m.game_invite_sent());
    } catch {
      toast.error(m.social_action_failed());
    }
    setPendingUserIds((current) => {
      const next = new Set(current);
      next.delete(friend.id);
      return next;
    });
  }

  async function unfriend(friend: SocialUser) {
    if (!onUnfriend) return;
    setPendingUserIds((current) => new Set(current).add(friend.id));
    try {
      await onUnfriend(friend.id);
      toast.success(m.friend_removed());
    } catch {
      toast.error(m.social_action_failed());
    }
    setPendingUserIds((current) => {
      const next = new Set(current);
      next.delete(friend.id);
      return next;
    });
  }

  async function spectate(friend: SocialUser) {
    if (!onSpectate || !friend.activeGame) return;
    setPendingUserIds((current) => new Set(current).add(friend.id));
    try {
      await onSpectate(friend.id);
      setOpenMobile(false);
      await navigate({ to: "/" });
    } catch {
      toast.error(m.social_action_failed());
    }
    setPendingUserIds((current) => {
      const next = new Set(current);
      next.delete(friend.id);
      return next;
    });
  }

  return (
    <SidebarGroup className="min-h-0 flex-1 overflow-hidden pt-0">
      <SidebarGroupLabel>{m.friends()}</SidebarGroupLabel>
      <SidebarGroupContent
        className="no-scrollbar min-h-0 flex-1 overflow-y-auto pr-1 group-data-[collapsible=icon]:pr-0"
        aria-busy={isLoading}
      >
        {isLoading ? (
          <SidebarMenu aria-hidden="true">
            {Array.from({ length: 3 }, (_, index) => (
              <SidebarMenuItem key={index}>
                <Skeleton className="h-9 w-full rounded-xl group-data-[collapsible=icon]:size-8" />
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        ) : sortedFriends.length === 0 ? (
          <Empty className="p-3 group-data-[collapsible=icon]:hidden">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HugeiconsIcon icon={UserGroupIcon} />
              </EmptyMedia>
              <EmptyTitle>{m.no_friends()}</EmptyTitle>
              <EmptyDescription className="text-xs">{m.no_friends_description()}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <SidebarMenu>
            {sortedFriends.map((friend) => {
              const pending = pendingUserIds.has(friend.id);
              const inviteEnabled =
                Boolean(onInvite) &&
                canInvite &&
                friend.online &&
                !friend.activeGame &&
                !unavailable.has(friend.id);
              return (
                <SidebarMenuItem key={friend.id}>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <SidebarMenuButton
                          type="button"
                          className="group-data-[collapsible=icon]:p-1!"
                          aria-label={m.open_friend_menu({ name: friend.name })}
                          tooltip={friend.name}
                        />
                      }
                    >
                      <Avatar size="sm">
                        {friend.imageUrl ? (
                          <AvatarImage src={friend.imageUrl} alt={friend.name} />
                        ) : null}
                        <AvatarFallback>{getUserInitials(friend.name)}</AvatarFallback>
                        <AvatarBadge
                          title={friend.online ? m.friend_online() : m.friend_offline()}
                          className={cn(
                            "ring-sidebar",
                            friend.online ? "bg-primary" : "bg-destructive",
                          )}
                        />
                      </Avatar>
                      <div className="flex min-w-0 flex-1 items-center justify-between gap-1.5 group-data-[collapsible=icon]:hidden">
                        <span className="min-w-0 truncate">{friend.name}</span>
                        {friend.activeGame ? (
                          <span
                            className="flex shrink-0 items-center gap-1 whitespace-nowrap text-xs text-muted-foreground tabular-nums"
                            aria-label={m.friend_in_game_duration({
                              duration: formatGameDuration(friend.activeGame.startedAt, now),
                            })}
                          >
                            <span aria-hidden="true">{m.friend_in_game()}</span>
                            <span aria-hidden="true" className="opacity-50">
                              ·
                            </span>
                            <span aria-hidden="true">
                              {formatGameDuration(friend.activeGame.startedAt, now)}
                            </span>
                          </span>
                        ) : null}
                      </div>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      align="start"
                      side={isMobile ? "bottom" : "right"}
                      className="w-52"
                    >
                      <DropdownMenuGroup>
                        <DropdownMenuLabel>{friend.name}</DropdownMenuLabel>
                        {friend.activeGame ? (
                          <DropdownMenuItem
                            disabled={!onSpectate || !canSpectate || pending}
                            onClick={() => void spectate(friend)}
                          >
                            <HugeiconsIcon icon={ViewIcon} />
                            {m.watch_game()}
                          </DropdownMenuItem>
                        ) : null}
                        <DropdownMenuItem
                          disabled={!inviteEnabled || pending}
                          onClick={() => void invite(friend)}
                        >
                          <HugeiconsIcon icon={GameController03Icon} />
                          {m.invite_to_game()}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          render={
                            <Link
                              to="/players/$playerId"
                              params={{ playerId: friend.id }}
                              onClick={() => setOpenMobile(false)}
                            />
                          }
                        >
                          <HugeiconsIcon icon={UserIcon} />
                          {m.view_profile()}
                        </DropdownMenuItem>
                      </DropdownMenuGroup>
                      {onUnfriend ? (
                        <>
                          <DropdownMenuSeparator />
                          <DropdownMenuGroup>
                            <DropdownMenuItem
                              variant="destructive"
                              disabled={pending}
                              onClick={() => void unfriend(friend)}
                            >
                              <HugeiconsIcon icon={UserRemove01Icon} />
                              {m.unfriend()}
                            </DropdownMenuItem>
                          </DropdownMenuGroup>
                        </>
                      ) : null}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </SidebarMenuItem>
              );
            })}
          </SidebarMenu>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

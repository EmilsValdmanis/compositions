import {
  GameController03Icon,
  UserGroupIcon,
  UserIcon,
  UserRemove01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
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

export function SidebarFriendsList({
  friends,
  isLoading = false,
  canInvite,
  unavailableUserIds = [],
  onInvite,
  onUnfriend,
}: {
  friends: SocialUser[];
  isLoading?: boolean;
  canInvite: boolean;
  unavailableUserIds?: string[];
  onInvite?: (userId: string) => Promise<unknown>;
  onUnfriend?: (userId: string) => Promise<unknown>;
}) {
  const { isMobile, setOpenMobile } = useSidebar();
  const [pendingUserIds, setPendingUserIds] = useState<Set<string>>(new Set());
  const unavailable = new Set(unavailableUserIds);
  const sortedFriends = [...friends].sort(
    (left, right) =>
      Number(right.online) - Number(left.online) || left.name.localeCompare(right.name),
  );

  async function invite(friend: SocialUser) {
    if (!onInvite) return;
    setPendingUserIds((current) => new Set(current).add(friend.id));
    try {
      await onInvite(friend.id);
      toast.success(m.game_invite_sent());
    } catch {
      toast.error(m.social_action_failed());
    } finally {
      setPendingUserIds((current) => {
        const next = new Set(current);
        next.delete(friend.id);
        return next;
      });
    }
  }

  async function unfriend(friend: SocialUser) {
    if (!onUnfriend) return;
    setPendingUserIds((current) => new Set(current).add(friend.id));
    try {
      await onUnfriend(friend.id);
      toast.success(m.friend_removed());
    } catch {
      toast.error(m.social_action_failed());
    } finally {
      setPendingUserIds((current) => {
        const next = new Set(current);
        next.delete(friend.id);
        return next;
      });
    }
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
                Boolean(onInvite) && canInvite && friend.online && !unavailable.has(friend.id);
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
                            friend.online ? "bg-primary" : "bg-muted-foreground",
                          )}
                        />
                      </Avatar>
                      <span>{friend.name}</span>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      align="start"
                      side={isMobile ? "bottom" : "right"}
                      className="w-52"
                    >
                      <DropdownMenuGroup>
                        <DropdownMenuLabel>{friend.name}</DropdownMenuLabel>
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

import { GameController03Icon, UserGroupIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import { type SocialUser } from "#/components/game-websocket-provider";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
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
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
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
}: {
  friends: SocialUser[];
  isLoading?: boolean;
  canInvite: boolean;
  unavailableUserIds?: string[];
  onInvite?: (userId: string) => Promise<unknown>;
}) {
  const unavailable = new Set(unavailableUserIds);
  const sortedFriends = [...friends].sort(
    (left, right) =>
      Number(right.online) - Number(left.online) || left.name.localeCompare(right.name),
  );

  async function invite(friend: SocialUser) {
    if (!onInvite) return;
    try {
      await onInvite(friend.id);
      toast.success(m.game_invite_sent());
    } catch {
      toast.error(m.social_action_failed());
    }
  }

  return (
    <SidebarGroup className="min-h-0 flex-1 overflow-hidden pt-0 group-data-[collapsible=icon]:hidden">
      <SidebarGroupLabel>{m.friends()}</SidebarGroupLabel>
      <SidebarGroupContent className="min-h-0 flex-1 overflow-y-auto pr-1" aria-busy={isLoading}>
        {isLoading ? (
          <SidebarMenu aria-hidden="true">
            {["w-20", "w-28", "w-24"].map((nameWidth) => (
              <SidebarMenuItem key={nameWidth}>
                <SidebarMenuButton render={<div />}>
                  <Skeleton className="size-6 shrink-0 rounded-full" />
                  <Skeleton className={cn("h-3", nameWidth)} />
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        ) : sortedFriends.length === 0 ? (
          <Empty className="p-3">
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
              const inviteEnabled = canInvite && friend.online && !unavailable.has(friend.id);
              return (
                <SidebarMenuItem key={friend.id}>
                  <SidebarMenuButton render={<div />}>
                    <Avatar size="sm">
                      {friend.imageUrl ? (
                        <AvatarImage src={friend.imageUrl} alt={friend.name} />
                      ) : null}
                      <AvatarFallback>{getUserInitials(friend.name)}</AvatarFallback>
                      <AvatarBadge
                        title={friend.online ? m.friend_online() : m.friend_offline()}
                        className={cn(friend.online ? "bg-primary" : "bg-muted-foreground")}
                      />
                    </Avatar>
                    <span>{friend.name}</span>
                  </SidebarMenuButton>
                  {inviteEnabled ? (
                    <SidebarMenuAction
                      type="button"
                      aria-label={m.invite_to_game()}
                      onClick={() => void invite(friend)}
                    >
                      <HugeiconsIcon icon={GameController03Icon} />
                    </SidebarMenuAction>
                  ) : null}
                </SidebarMenuItem>
              );
            })}
          </SidebarMenu>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

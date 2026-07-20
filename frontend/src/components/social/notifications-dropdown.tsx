import {
  Cancel01Icon,
  GameController03Icon,
  Notification02Icon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "#/components/ui/item";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "#/components/ui/popover";
import { SidebarMenuButton } from "#/components/ui/sidebar";
import { cn } from "#/lib/utils";
import { getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

export function NotificationsDropdown({
  presentation = "button",
}: {
  presentation?: "button" | "sidebar";
}) {
  const { state, respondFriendRequest, respondGameInvite } = useGameWebSocket();
  const navigate = useNavigate();
  const [pendingIds, setPendingIds] = useState<Set<string>>(new Set());
  const [currentTime, setCurrentTime] = useState(() => Date.now());
  const friendRequests = state.social.incomingFriendRequests;
  const gameInvites = state.social.gameInvites.filter(
    (invite) => new Date(invite.expiresAt).getTime() > currentTime,
  );
  const notificationCount = friendRequests.length + gameInvites.length;

  useEffect(() => {
    const now = Date.now();
    const nextExpiry = state.social.gameInvites.reduce<number | null>((earliest, invite) => {
      const expiry = new Date(invite.expiresAt).getTime();
      if (!Number.isFinite(expiry) || expiry <= now) return earliest;
      return earliest === null || expiry < earliest ? expiry : earliest;
    }, null);
    if (nextExpiry === null) return;

    const timeout = window.setTimeout(
      () => setCurrentTime(Date.now()),
      Math.max(0, nextExpiry - now + 1),
    );
    return () => window.clearTimeout(timeout);
  }, [currentTime, state.social.gameInvites]);

  async function respond(id: string, action: () => Promise<unknown>, successMessage: string) {
    setPendingIds((current) => new Set(current).add(id));
    try {
      await action();
      toast.success(successMessage);
    } catch {
      toast.error(m.social_action_failed());
    } finally {
      setPendingIds((current) => {
        const next = new Set(current);
        next.delete(id);
        return next;
      });
    }
  }

  async function acceptGameInvite(inviteId: string) {
    const result = await respondGameInvite(inviteId, true);
    await navigate({ to: "/" });
    return result;
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          presentation === "sidebar" ? (
            <SidebarMenuButton tooltip={m.notifications()} className="relative" />
          ) : (
            <Button
              variant="outline"
              size="icon"
              aria-label={m.notifications()}
              className="relative"
            />
          )
        }
      >
        <HugeiconsIcon icon={Notification02Icon} />
        {presentation === "sidebar" ? <span>{m.notifications()}</span> : null}
        {notificationCount > 0 ? (
          <Badge
            variant="destructive"
            className={cn(
              "h-5 min-w-5 justify-center px-1 text-[0.65rem]",
              presentation === "sidebar"
                ? "ml-auto group-data-[collapsible=icon]:hidden"
                : "absolute -top-1 -right-1",
            )}
          >
            {notificationCount > 9 ? "9+" : notificationCount}
          </Badge>
        ) : null}
      </PopoverTrigger>
      <PopoverContent
        align={presentation === "sidebar" ? "start" : "end"}
        side={presentation === "sidebar" ? "right" : "bottom"}
        className="max-h-[min(32rem,var(--available-height))] w-88 overflow-y-auto"
      >
        <PopoverHeader>
          <PopoverTitle>{m.notifications()}</PopoverTitle>
          <PopoverDescription>{m.notifications_description()}</PopoverDescription>
        </PopoverHeader>

        {notificationCount === 0 ? (
          <Empty className="p-6">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HugeiconsIcon icon={Notification02Icon} />
              </EmptyMedia>
              <EmptyTitle>{m.no_notifications()}</EmptyTitle>
              <EmptyDescription>{m.no_notifications_description()}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ItemGroup className="gap-2">
            {friendRequests.map((request) => {
              const pending = pendingIds.has(request.id);
              return (
                <Item key={request.id} variant="muted" size="sm">
                  <ItemMedia>
                    <Avatar>
                      {request.user.imageUrl ? (
                        <AvatarImage src={request.user.imageUrl} alt={request.user.name} />
                      ) : null}
                      <AvatarFallback>{getUserInitials(request.user.name)}</AvatarFallback>
                    </Avatar>
                  </ItemMedia>
                  <ItemContent>
                    <ItemTitle>{m.friend_request()}</ItemTitle>
                    <ItemDescription>
                      {m.friend_request_from({ name: request.user.name })}
                    </ItemDescription>
                  </ItemContent>
                  <ItemActions>
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="outline"
                      disabled={pending}
                      aria-label={m.decline()}
                      onClick={() =>
                        void respond(
                          request.id,
                          () => respondFriendRequest(request.id, false),
                          m.friend_request_declined(),
                        )
                      }
                    >
                      <HugeiconsIcon icon={Cancel01Icon} />
                    </Button>
                    <Button
                      type="button"
                      size="icon-sm"
                      disabled={pending}
                      aria-label={m.accept()}
                      onClick={() =>
                        void respond(
                          request.id,
                          () => respondFriendRequest(request.id, true),
                          m.friend_request_accepted(),
                        )
                      }
                    >
                      <HugeiconsIcon icon={Tick02Icon} />
                    </Button>
                  </ItemActions>
                </Item>
              );
            })}

            {gameInvites.map((invite) => {
              const pending = pendingIds.has(invite.id);
              return (
                <Item key={invite.id} variant="muted" size="sm">
                  <ItemMedia>
                    <Avatar>
                      {invite.user.imageUrl ? (
                        <AvatarImage src={invite.user.imageUrl} alt={invite.user.name} />
                      ) : null}
                      <AvatarFallback>{getUserInitials(invite.user.name)}</AvatarFallback>
                    </Avatar>
                  </ItemMedia>
                  <ItemContent>
                    <ItemTitle>
                      <HugeiconsIcon icon={GameController03Icon} />
                      {m.game_invite()}
                    </ItemTitle>
                    <ItemDescription>
                      {m.game_invite_from({ name: invite.user.name, roomCode: invite.roomCode })}
                    </ItemDescription>
                  </ItemContent>
                  <ItemActions>
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="outline"
                      disabled={pending}
                      aria-label={m.decline()}
                      onClick={() =>
                        void respond(
                          invite.id,
                          () => respondGameInvite(invite.id, false),
                          m.game_invite_declined(),
                        )
                      }
                    >
                      <HugeiconsIcon icon={Cancel01Icon} />
                    </Button>
                    <Button
                      type="button"
                      size="icon-sm"
                      disabled={pending}
                      aria-label={m.accept()}
                      onClick={() =>
                        void respond(
                          invite.id,
                          () => acceptGameInvite(invite.id),
                          m.game_invite_accepted(),
                        )
                      }
                    >
                      <HugeiconsIcon icon={Tick02Icon} />
                    </Button>
                  </ItemActions>
                </Item>
              );
            })}
          </ItemGroup>
        )}
      </PopoverContent>
    </Popover>
  );
}

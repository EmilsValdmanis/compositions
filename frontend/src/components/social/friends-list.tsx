import { GameController03Icon, UserGroupIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import { type SocialUser } from "#/components/game-websocket-provider";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
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
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "#/components/ui/item";
import { getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

export function FriendsList({
  friends,
  canInvite,
  unavailableUserIds = [],
  onInvite,
}: {
  friends: SocialUser[];
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
    <Card className="min-w-0 border border-border/70 shadow-sm">
      <CardHeader>
        <CardTitle>{m.friends()}</CardTitle>
        <CardDescription>{m.friends_description()}</CardDescription>
      </CardHeader>
      <CardContent>
        {sortedFriends.length === 0 ? (
          <Empty className="p-6">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HugeiconsIcon icon={UserGroupIcon} />
              </EmptyMedia>
              <EmptyTitle>{m.no_friends()}</EmptyTitle>
              <EmptyDescription>{m.no_friends_description()}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ItemGroup className="gap-2">
            {sortedFriends.map((friend) => {
              const inviteEnabled = canInvite && friend.online && !unavailable.has(friend.id);
              return (
                <Item
                  key={friend.id}
                  variant="muted"
                  size="sm"
                  render={
                    inviteEnabled ? (
                      <button
                        type="button"
                        aria-label={m.invite_to_game()}
                        onClick={() => void invite(friend)}
                      />
                    ) : undefined
                  }
                >
                  <ItemMedia>
                    <Avatar>
                      {friend.imageUrl ? (
                        <AvatarImage src={friend.imageUrl} alt={friend.name} />
                      ) : null}
                      <AvatarFallback>{getUserInitials(friend.name)}</AvatarFallback>
                      <AvatarBadge
                        className={friend.online ? "bg-primary" : "bg-muted-foreground"}
                      />
                    </Avatar>
                  </ItemMedia>
                  <ItemContent>
                    <ItemTitle>{friend.name}</ItemTitle>
                  </ItemContent>
                  <ItemActions>
                    {inviteEnabled ? <HugeiconsIcon icon={GameController03Icon} /> : null}
                    <Badge variant={friend.online ? "secondary" : "outline"}>
                      {friend.online ? m.friend_online() : m.friend_offline()}
                    </Badge>
                  </ItemActions>
                </Item>
              );
            })}
          </ItemGroup>
        )}
      </CardContent>
    </Card>
  );
}

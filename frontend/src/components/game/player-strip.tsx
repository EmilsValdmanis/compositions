import {
  Cards02Icon,
  SkullIcon,
  UserAdd01Icon,
  UserIcon,
  UserMultipleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { toast } from "sonner";
import {
  type GameSnapshot,
  type PlayerSnapshot,
  type SocialState,
} from "#/components/game-websocket-provider";
import { PlayerEmoteBubble } from "#/components/game/player-emotes";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "#/components/ui/item";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

function PlayerAvatar({
  player,
  compactOnMobile = false,
  currentPlayerId,
  social,
  onSendFriendRequest,
}: {
  player: PlayerSnapshot;
  compactOnMobile?: boolean;
  currentPlayerId?: string;
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
}) {
  const isSelf = player.playerId === currentPlayerId || player.userId === social?.userId;
  const isFriend = social?.friends.some((friend) => friend.id === player.userId) ?? false;
  const requestSent = social?.outgoingFriendRequestUserIds.includes(player.userId ?? "") ?? false;
  const requestReceived =
    social?.incomingFriendRequests.some((request) => request.user.id === player.userId) ?? false;

  async function addFriend() {
    if (!player.userId || !onSendFriendRequest) return;
    try {
      await onSendFriendRequest(player.userId);
      toast.success(m.friend_request_sent());
    } catch {
      toast.error(m.social_action_failed());
    }
  }

  const avatar = (
    <Avatar className={cn("shrink-0", compactOnMobile ? "size-8 xl:size-10" : null)}>
      {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
      <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
      <AvatarBadge
        className={cn("ring-card", player.connected ? "bg-primary" : "bg-destructive")}
      />
    </Avatar>
  );

  if (!player.userId) return avatar;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="rounded-full"
            aria-label={m.open_player_menu({ name: player.name })}
          />
        }
      >
        {avatar}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-52">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{player.name}</DropdownMenuLabel>
          <DropdownMenuItem
            render={<Link to="/players/$playerId" params={{ playerId: player.userId }} />}
          >
            <HugeiconsIcon icon={UserIcon} />
            {m.view_profile()}
          </DropdownMenuItem>
          {!isSelf && onSendFriendRequest ? (
            <DropdownMenuItem
              disabled={isFriend || requestSent || requestReceived}
              onClick={() => void addFriend()}
            >
              <HugeiconsIcon icon={isFriend ? UserMultipleIcon : UserAdd01Icon} />
              {isFriend
                ? m.already_friends()
                : requestSent
                  ? m.request_sent()
                  : requestReceived
                    ? m.friend_request_received()
                    : m.add_friend()}
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function PlayerStrip({
  players,
  game,
  showHostBadges = true,
  showTurnIndicator = true,
  mobileHorizontal = false,
  presentation = "default",
  emoteClassName,
  currentPlayerId,
  social,
  onSendFriendRequest,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  showHostBadges?: boolean;
  showTurnIndicator?: boolean;
  mobileHorizontal?: boolean;
  presentation?: "default" | "menu";
  emoteClassName?: string;
  currentPlayerId?: string;
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
}) {
  const playerStates = game?.players ?? [];
  const isMenu = presentation === "menu";
  return (
    <ItemGroup
      data-presentation={presentation}
      className={cn(
        "grid min-w-0 max-w-full gap-2 has-data-[size=sm]:gap-2",
        isMenu
          ? "gap-0.5 overflow-visible p-0"
          : mobileHorizontal
            ? "grid-flow-col auto-cols-[minmax(9rem,1fr)] overflow-x-auto overscroll-x-contain pb-1 xl:grid-flow-row xl:auto-cols-auto xl:overflow-visible xl:pb-0"
            : "min-h-0 flex-1 content-start overflow-y-auto overscroll-y-contain pr-1",
      )}
    >
      {players.map((player) => {
        const gamePlayer = playerStates.find((item) => item.playerId === player.playerId);
        const isTurn = game?.turn.playerId === player.playerId;
        const showActiveTurn = showTurnIndicator && isTurn && !player.forfeited;
        return (
          <Item
            key={player.playerId}
            role="listitem"
            aria-current={showActiveTurn ? "true" : undefined}
            data-active-turn={showActiveTurn ? "true" : undefined}
            variant={isMenu ? "default" : "outline"}
            size="xs"
            className={cn(
              "player-turn-surface relative min-w-0 max-w-full flex-nowrap",
              showActiveTurn && "player-turn-surface-active",
              isMenu
                ? "gap-2 rounded-xl in-data-[slot=dropdown-menu-content]:px-1.5 in-data-[slot=dropdown-menu-content]:py-1"
                : mobileHorizontal
                  ? "gap-2 px-2 py-1.5 xl:gap-3 xl:px-3 xl:py-2"
                  : null,
            )}
          >
            {player.activeEmote ? (
              <PlayerEmoteBubble emote={player.activeEmote} className={emoteClassName} />
            ) : null}
            <ItemMedia>
              <PlayerAvatar
                player={player}
                compactOnMobile={mobileHorizontal || isMenu}
                currentPlayerId={currentPlayerId}
                social={social}
                onSendFriendRequest={onSendFriendRequest}
              />
            </ItemMedia>
            <ItemContent className="min-w-0">
              <ItemTitle className="w-full" title={player.name}>
                {player.name}
                {showActiveTurn ? (
                  <span
                    role="status"
                    aria-label={m.player_turn({ name: player.name })}
                    className="sr-only"
                  />
                ) : null}
              </ItemTitle>
            </ItemContent>
            <ItemActions className="min-w-max shrink-0 flex-nowrap justify-end gap-1.5">
              {player.forfeited ? (
                <HugeiconsIcon
                  icon={SkullIcon}
                  className="size-5 shrink-0 text-destructive"
                  aria-label={m.player_forfeited({ name: player.name })}
                />
              ) : null}
              {showHostBadges && player.isHost ? (
                <Badge variant="secondary">{m.host()}</Badge>
              ) : null}
              {gamePlayer ? (
                <Badge
                  variant={showActiveTurn ? "default" : "outline"}
                  aria-label={m.cards_count({ count: gamePlayer.handCount })}
                  data-card-motion-player={player.playerId}
                >
                  <AnimatedNumber value={gamePlayer.handCount} />
                  <HugeiconsIcon icon={Cards02Icon} aria-hidden="true" />
                </Badge>
              ) : null}
            </ItemActions>
          </Item>
        );
      })}
    </ItemGroup>
  );
}

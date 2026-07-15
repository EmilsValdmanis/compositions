import { Cards02Icon, SkullIcon, UserIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
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
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Spinner } from "#/components/ui/spinner";
import { P } from "#/components/typography";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

function PlayerAvatar({
  player,
  compactOnMobile = false,
}: {
  player: PlayerSnapshot;
  compactOnMobile?: boolean;
}) {
  const avatar = (
    <Avatar className={cn("shrink-0", compactOnMobile ? "size-8 xl:size-10" : null)}>
      {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
      <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
      <AvatarBadge
        className={cn("ring-border", player.connected ? "bg-primary" : "bg-destructive")}
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
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  showHostBadges?: boolean;
  showTurnIndicator?: boolean;
  mobileHorizontal?: boolean;
}) {
  const playerStates = game?.players ?? [];
  return (
    <div
      className={cn(
        "grid w-full min-w-0 max-w-full gap-2",
        mobileHorizontal
          ? "grid-flow-col auto-cols-[minmax(9rem,1fr)] overflow-x-auto overscroll-x-contain pb-1 xl:grid-flow-row xl:auto-cols-auto xl:overflow-visible xl:pb-0"
          : "min-h-0 flex-1 content-start overflow-y-auto overscroll-y-contain pr-1",
      )}
    >
      {players.map((player) => {
        const gamePlayer = playerStates.find((item) => item.playerId === player.playerId);
        const isTurn = game?.turn.playerId === player.playerId;
        const showActiveTurn = showTurnIndicator && isTurn && !player.forfeited;
        return (
          <div
            key={player.playerId}
            className={cn(
              "relative grid w-full min-w-0 max-w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-3xl border px-3 py-2",
              mobileHorizontal ? "gap-2 px-2 py-1.5 xl:gap-3 xl:px-3 xl:py-2" : null,
              showActiveTurn ? "border-primary/40 bg-primary/10" : "border-border/60 bg-muted/20",
            )}
          >
            {player.activeEmote ? <PlayerEmoteBubble emote={player.activeEmote} /> : null}
            <PlayerAvatar player={player} compactOnMobile={mobileHorizontal} />
            <P
              size="sm"
              className="block min-w-0 max-w-full overflow-hidden text-ellipsis whitespace-nowrap font-medium"
              title={player.name}
            >
              {player.name}
            </P>
            <div className="flex min-w-max shrink-0 flex-nowrap items-center justify-end gap-1.5">
              {player.forfeited ? (
                <HugeiconsIcon
                  icon={SkullIcon}
                  className="size-5 shrink-0 text-destructive"
                  aria-label={m.player_forfeited({ name: player.name })}
                />
              ) : showActiveTurn ? (
                <Spinner
                  className="size-5 shrink-0 text-primary"
                  aria-label={m.player_turn({ name: player.name })}
                />
              ) : null}
              {showHostBadges && player.isHost ? (
                <Badge variant="secondary">{m.host()}</Badge>
              ) : null}
              {gamePlayer ? (
                <Badge
                  variant="outline"
                  aria-label={m.cards_count({ count: gamePlayer.handCount })}
                  data-card-motion-player={player.playerId}
                >
                  <AnimatedNumber value={gamePlayer.handCount} />
                  <HugeiconsIcon icon={Cards02Icon} aria-hidden="true" />
                </Badge>
              ) : null}
            </div>
          </div>
        );
      })}
    </div>
  );
}

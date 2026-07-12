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
import { cn, getUserInitials } from "#/lib/utils";

function PlayerAvatar({ player }: { player: PlayerSnapshot }) {
  const avatar = (
    <Avatar className="shrink-0">
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
            aria-label={`Open ${player.name}'s player menu`}
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
            View profile
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
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  showHostBadges?: boolean;
  showTurnIndicator?: boolean;
}) {
  const playerStates = game?.players ?? [];
  return (
    <div className="grid w-full min-w-0 max-w-full gap-2">
      {players.map((player) => {
        const gamePlayer = playerStates.find((item) => item.playerId === player.playerId);
        const isTurn = game?.turn.playerId === player.playerId;
        const showActiveTurn = showTurnIndicator && isTurn && !player.forfeited;
        return (
          <div
            key={player.playerId}
            className={cn(
              "relative grid w-full min-w-0 max-w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-3xl border px-3 py-2",
              showActiveTurn ? "border-primary/40 bg-primary/10" : "border-border/60 bg-muted/20",
            )}
          >
            {player.activeEmote ? <PlayerEmoteBubble emote={player.activeEmote} /> : null}
            <PlayerAvatar player={player} />
            <p
              className="block min-w-0 max-w-full overflow-hidden text-ellipsis whitespace-nowrap font-medium"
              title={player.name}
            >
              {player.name}
            </p>
            <div className="flex min-w-max shrink-0 flex-nowrap items-center justify-end gap-1.5">
              {player.forfeited ? (
                <HugeiconsIcon
                  icon={SkullIcon}
                  className="size-5 shrink-0 text-destructive"
                  aria-label={`${player.name} forfeited`}
                />
              ) : showActiveTurn ? (
                <Spinner
                  className="size-5 shrink-0 text-primary"
                  aria-label={`${player.name}'s turn`}
                />
              ) : null}
              {showHostBadges && player.isHost ? <Badge variant="secondary">Host</Badge> : null}
              {gamePlayer ? (
                <Badge variant="outline" aria-label={`${gamePlayer.handCount} cards`}>
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

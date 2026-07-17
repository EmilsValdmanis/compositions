import { Add01Icon, CardExchange01Icon, SparklesIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { playerById } from "#/components/game/game-view-helpers";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Caption } from "#/components/typography";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

function PlayerMarker({
  players,
  playerId,
  className,
}: {
  players: PlayerSnapshot[];
  playerId?: string;
  className?: string;
}) {
  const player = playerById(players, playerId);

  if (!player) {
    return null;
  }

  return (
    <Avatar size="sm" className={className}>
      {player.imageUrl ? <AvatarImage src={player.imageUrl} alt={player.name} /> : null}
      <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
    </Avatar>
  );
}

export function ActivityLabel({
  players,
  playerId,
  label = m.activity_new(),
  icon,
  iconOnly = false,
  className,
  offsetClassName,
}: {
  players: PlayerSnapshot[];
  playerId?: string;
  label?: string;
  icon?: React.ComponentProps<typeof HugeiconsIcon>["icon"];
  iconOnly?: boolean;
  className?: string;
  offsetClassName?: string;
}) {
  return (
    <Caption
      className={cn(
        "flex h-5 items-center gap-1 text-[0.6rem]/4 font-medium tracking-wide uppercase md:text-[0.65rem]/4",
        offsetClassName,
        className,
      )}
      title={iconOnly ? label : undefined}
    >
      {icon ? (
        <HugeiconsIcon
          icon={icon}
          className={cn("size-3.5", iconOnly && "size-4")}
          strokeWidth={2}
          aria-hidden="true"
        />
      ) : null}
      <span className={cn(iconOnly && "sr-only")}>{label}</span>
      {playerId ? (
        <PlayerMarker players={players} playerId={playerId} className="size-4 shrink-0" />
      ) : null}
    </Caption>
  );
}

export function NewActivityLabel(
  props: Omit<Parameters<typeof ActivityLabel>[0], "label" | "icon">,
) {
  return <ActivityLabel {...props} label={m.activity_new()} icon={SparklesIcon} />;
}

export function AddActivityLabel(
  props: Omit<Parameters<typeof ActivityLabel>[0], "label" | "icon" | "iconOnly">,
) {
  return <ActivityLabel {...props} label={m.activity_add()} icon={Add01Icon} iconOnly />;
}

export function ReclaimActivityLabel(
  props: Omit<Parameters<typeof ActivityLabel>[0], "label" | "icon" | "iconOnly">,
) {
  return (
    <ActivityLabel {...props} label={m.activity_reclaim()} icon={CardExchange01Icon} iconOnly />
  );
}

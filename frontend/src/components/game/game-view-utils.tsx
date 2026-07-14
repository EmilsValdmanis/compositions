import { SparklesIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type PlayerSnapshot } from "#/components/game-websocket-provider";
import { playerById } from "#/components/game/game-view-helpers";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Text } from "#/components/typography";
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
  className,
  offsetClassName,
}: {
  players: PlayerSnapshot[];
  playerId?: string;
  label?: string;
  icon?: React.ComponentProps<typeof HugeiconsIcon>["icon"];
  className?: string;
  offsetClassName?: string;
}) {
  return (
    <Text
      as="div"
      variant="activity"
      className={cn("flex h-5 items-center gap-1", offsetClassName, className)}
    >
      {icon ? <HugeiconsIcon icon={icon} className="size-3.5" strokeWidth={2} /> : null}
      <span>{label}</span>
      {playerId ? (
        <PlayerMarker players={players} playerId={playerId} className="size-4 shrink-0" />
      ) : null}
    </Text>
  );
}

export function NewActivityLabel(
  props: Omit<Parameters<typeof ActivityLabel>[0], "label" | "icon">,
) {
  return <ActivityLabel {...props} label={m.activity_new()} icon={SparklesIcon} />;
}

export function ReclaimActivityLabel(
  props: Omit<Parameters<typeof ActivityLabel>[0], "label" | "icon">,
) {
  return <ActivityLabel {...props} label={m.activity_reclaim()} />;
}

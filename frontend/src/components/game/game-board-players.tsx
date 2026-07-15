import { ChevronDownIcon, UndoIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { PlayerEmotePicker } from "#/components/game/player-emotes";
import { PlayerStrip } from "#/components/game/player-strip";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Button } from "#/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { Spinner } from "#/components/ui/spinner";
import { cn } from "#/lib/utils";
import { Badge } from "../ui/badge";
import { m } from "#/paraglide/messages.js";

export function GameBoardPlayers({
  players,
  game,
  connectedPlayers,
  hasDraftedCompositions,
  showTurnIndicator = true,
  compactOnMobile = false,
  onResetDraftCompositions,
  onSendEmote,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  connectedPlayers: number;
  hasDraftedCompositions: boolean;
  showTurnIndicator?: boolean;
  compactOnMobile?: boolean;
  onResetDraftCompositions: () => void;
  onSendEmote: (emoji: string) => void;
}) {
  const activePlayerCount = players.filter((player) => !player.forfeited).length;
  const turnPlayer = players.find((player) => player.playerId === game?.turn.playerId);
  return (
    <Card
      size="sm"
      className={cn(
        "min-w-0 shrink-0 overflow-hidden [--card-spacing:--spacing(2)] xl:grow xl:overflow-y-auto xl:[--card-spacing:--spacing(4)]",
        compactOnMobile ? "py-2! xl:py-4!" : "[@media(max-height:600px)]:h-full",
      )}
    >
      {compactOnMobile ? (
        <CardHeader className="xl:hidden">
          <div className="flex min-w-0 items-center gap-1">
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    className="h-10 min-w-0 flex-1 justify-start px-2"
                  />
                }
              >
                {turnPlayer ? (
                  <>
                    <Spinner className="size-4 shrink-0 text-primary" aria-hidden="true" />
                    <span className="truncate font-medium">{turnPlayer.name}</span>
                  </>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
                <HugeiconsIcon icon={ChevronDownIcon} className="ml-auto size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="start"
                sideOffset={8}
                className="w-[min(24rem,calc(100vw-2rem))] p-2"
              >
                <PlayerStrip
                  players={players}
                  game={game}
                  showHostBadges={false}
                  showTurnIndicator={showTurnIndicator}
                />

                {hasDraftedCompositions ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={onResetDraftCompositions}
                    className="mt-2 w-full"
                    size="lg"
                  >
                    <HugeiconsIcon icon={UndoIcon} strokeWidth={2} data-icon="inline-start" />
                    {m.reset()}
                  </Button>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>

            <PlayerEmotePicker onSendEmote={onSendEmote} />
          </div>
        </CardHeader>
      ) : null}

      <CardHeader className={compactOnMobile ? "hidden xl:grid" : undefined}>
        <CardTitle>{m.players()}</CardTitle>
        <CardAction>
          <div className="flex items-center gap-2">
            <PlayerEmotePicker onSendEmote={onSendEmote} />
            <Badge variant="outline">
              <AnimatedNumber value={connectedPlayers} />/
              <AnimatedNumber value={activePlayerCount} /> {m.online()}
            </Badge>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent
        className={cn("h-full min-w-0 flex-col gap-3", compactOnMobile ? "hidden xl:flex" : "flex")}
      >
        <PlayerStrip
          players={players}
          game={game}
          showHostBadges={false}
          showTurnIndicator={showTurnIndicator}
        />

        {hasDraftedCompositions ? (
          <Button
            type="button"
            variant="ghost"
            onClick={onResetDraftCompositions}
            className="mt-auto w-full"
            size="lg"
          >
            <HugeiconsIcon icon={UndoIcon} strokeWidth={2} data-icon="inline-start" />
            {m.reset()}
          </Button>
        ) : null}
      </CardContent>
    </Card>
  );
}

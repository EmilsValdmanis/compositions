import { ChevronDownIcon, RankingIcon, UndoIcon, ViewIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type GameSnapshot,
  type PlayerSnapshot,
  type SocialState,
} from "#/components/game-websocket-provider";
import { MobilePlayerEmotes, PlayerEmotePicker } from "#/components/game/player-emotes";
import { PlayerStrip } from "#/components/game/player-strip";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "#/components/ui/popover";
import { Spinner } from "#/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";
import { P } from "#/components/typography";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

function SpectatorCount({ count }: { count: number }) {
  if (count <= 0) return null;

  return (
    <Badge variant="outline" aria-label={`${m.spectators()}: ${count}`}>
      <HugeiconsIcon icon={ViewIcon} data-icon="inline-start" />
      <AnimatedNumber value={count} />
    </Badge>
  );
}

function GameScoreboard({
  game,
  players,
}: {
  game: GameSnapshot | null;
  players: PlayerSnapshot[];
}) {
  const rows = (game?.players ?? [])
    .toSorted((left, right) => left.totalPoints - right.totalPoints)
    .map((playerState, index) => ({
      rank: index + 1,
      playerState,
      player: players.find((player) => player.playerId === playerState.playerId),
    }));

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label={m.results()}
            disabled={!game}
          />
        }
      >
        <HugeiconsIcon icon={RankingIcon} strokeWidth={2} />
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[min(24rem,calc(100vw-2rem))] p-0">
        <PopoverHeader className="px-4 pt-4">
          <PopoverTitle className="flex items-center gap-2">
            <HugeiconsIcon
              icon={RankingIcon}
              className="size-4 text-muted-foreground"
              data-slot="scoreboard-title-icon"
              aria-hidden="true"
            />
            {m.results()}
          </PopoverTitle>
        </PopoverHeader>
        <div className="overflow-hidden rounded-b-3xl border-t border-border/70">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>#</TableHead>
                <TableHead>{m.player()}</TableHead>
                <TableHead className="text-right">{m.score()}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map(({ rank, player, playerState }) => (
                <TableRow key={playerState.playerId}>
                  <TableCell className="tabular-nums">{rank}</TableCell>
                  <TableCell>
                    <div className="flex min-w-0 items-center gap-2">
                      <Avatar size="sm">
                        {player?.imageUrl ? (
                          <AvatarImage src={player.imageUrl} alt={player.name} />
                        ) : null}
                        <AvatarFallback>
                          {getUserInitials(player?.name ?? m.unknown_player())}
                        </AvatarFallback>
                        {player ? (
                          <AvatarBadge
                            className={cn(
                              "ring-popover",
                              player.connected ? "bg-primary" : "bg-destructive",
                            )}
                          />
                        ) : null}
                      </Avatar>
                      <P size="sm" className="max-w-36 truncate font-medium" title={player?.name}>
                        {player?.name ?? m.unknown_player()}
                      </P>
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    <AnimatedNumber value={playerState.totalPoints} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function GameBoardPlayers({
  players,
  game,
  hasDraftedCompositions,
  showTurnIndicator = true,
  compactOnMobile = false,
  onResetDraftCompositions,
  onSendEmote,
  currentPlayerId,
  spectatorCount = 0,
  canSendEmote = true,
  social,
  onSendFriendRequest,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  connectedPlayers: number;
  hasDraftedCompositions: boolean;
  showTurnIndicator?: boolean;
  compactOnMobile?: boolean;
  onResetDraftCompositions: () => void;
  onSendEmote: (emoji: string) => void;
  currentPlayerId?: string;
  spectatorCount?: number;
  canSendEmote?: boolean;
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
}) {
  const turnPlayer = players.find((player) => player.playerId === game?.turn.playerId);
  return (
    <>
      {compactOnMobile && canSendEmote ? <MobilePlayerEmotes players={players} /> : null}
      <Card
        size="sm"
        className={cn(
          "min-w-0 shrink-0 overflow-hidden [--card-spacing:--spacing(2)] xl:grow xl:[--card-spacing:--spacing(4)]",
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
                <DropdownMenuContent>
                  <PlayerStrip
                    players={players}
                    game={game}
                    presentation="menu"
                    showHostBadges={false}
                    showTurnIndicator={showTurnIndicator}
                    emoteClassName="hidden"
                    currentPlayerId={currentPlayerId}
                    social={social}
                    onSendFriendRequest={onSendFriendRequest}
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

              <SpectatorCount count={spectatorCount} />
              {game ? <GameScoreboard game={game} players={players} /> : null}
              {canSendEmote ? <PlayerEmotePicker onSendEmote={onSendEmote} /> : null}
            </div>
          </CardHeader>
        ) : null}

        <CardHeader className={cn("items-center", compactOnMobile && "hidden xl:flex")}>
          <CardTitle>{m.players()}</CardTitle>
          <CardAction className="ml-auto">
            <div className="flex items-center gap-2">
              <SpectatorCount count={spectatorCount} />
              {canSendEmote ? <PlayerEmotePicker onSendEmote={onSendEmote} /> : null}
              {game ? <GameScoreboard game={game} players={players} /> : null}
            </div>
          </CardAction>
        </CardHeader>
        <CardContent
          className={cn(
            "min-h-0 min-w-0 flex-1 flex-col gap-3",
            compactOnMobile ? "hidden xl:flex" : "flex",
          )}
        >
          <PlayerStrip
            players={players}
            game={game}
            showHostBadges={false}
            showTurnIndicator={showTurnIndicator}
            emoteClassName={compactOnMobile ? "hidden xl:grid" : undefined}
            currentPlayerId={currentPlayerId}
            social={social}
            onSendFriendRequest={onSendFriendRequest}
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
    </>
  );
}

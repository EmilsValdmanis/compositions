import { Alert02Icon, ChampionIcon, RankingIcon, Refresh01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useMemo, useRef } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import { Spinner } from "#/components/ui/spinner";
import { TableBody, TableCell, TableHead, TableHeader, TableRow } from "#/components/ui/table";
import { H1, P } from "#/components/typography";
import { leaderboardInfiniteOptions, type LeaderboardPlayer } from "#/lib/leaderboard";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { getLocale, type Locale } from "#/paraglide/runtime.js";

const NUMBER_FORMATTERS: Record<Locale, Intl.NumberFormat> = {
  en: new Intl.NumberFormat("en"),
  lv: new Intl.NumberFormat("lv"),
};

const leaderboardGrid =
  "grid-cols-[2.75rem_minmax(0,1fr)_3.75rem_3.75rem] sm:grid-cols-[3rem_minmax(0,1fr)_4rem_4.5rem_6.5rem] lg:grid-cols-[3rem_minmax(12rem,1fr)_5rem_6rem_7rem_6rem_7rem]";

function formatNumber(value: number) {
  return NUMBER_FORMATTERS[getLocale()].format(value);
}

function formatPlaytime(totalSeconds: number) {
  const totalMinutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return hours > 0
    ? m.playtime_hours_minutes({ hours, minutes })
    : m.playtime_minutes({ minutes: totalMinutes });
}

function RankBadge({ rank }: { rank: number }) {
  return (
    <Badge
      variant={rank === 1 ? "default" : rank <= 3 ? "secondary" : "outline"}
      className="min-w-8 tabular-nums"
    >
      {rank}
    </Badge>
  );
}

function PlayerIdentity({ player, isCurrent }: { player: LeaderboardPlayer; isCurrent: boolean }) {
  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <Avatar size="sm" className="hidden sm:flex">
        {player.imageUrl ? <AvatarImage src={player.imageUrl} alt="" /> : null}
        <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
      </Avatar>
      <Link
        to="/players/$playerId"
        params={{ playerId: player.playerId }}
        className="min-w-0 truncate font-medium underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {player.name}
      </Link>
      {isCurrent ? (
        <Badge variant="secondary" className="hidden shrink-0 sm:inline-flex">
          {m.you()}
        </Badge>
      ) : null}
    </div>
  );
}

function LeaderboardHeader() {
  return (
    <TableHeader className="sticky top-0 z-10 grid bg-card shadow-sm">
      <TableRow className={cn("grid hover:bg-transparent", leaderboardGrid)}>
        <TableHead className="flex h-11 items-center px-2 sm:px-3">{m.rank()}</TableHead>
        <TableHead className="flex h-11 items-center px-2 sm:px-3">{m.player()}</TableHead>
        <TableHead className="flex h-11 items-center justify-end px-2 sm:px-3">
          {m.wins()}
        </TableHead>
        <TableHead className="flex h-11 items-center justify-end px-2 sm:px-3">
          {m.games_label()}
        </TableHead>
        <TableHead className="hidden h-11 items-center justify-end px-3 sm:flex">
          {m.total_playtime()}
        </TableHead>
        <TableHead className="hidden h-11 items-center justify-end px-3 lg:flex">
          {m.round_wins_label()}
        </TableHead>
        <TableHead className="hidden h-11 items-center justify-end px-3 lg:flex">
          {m.points_inflicted()}
        </TableHead>
      </TableRow>
    </TableHeader>
  );
}

function LeaderboardTableRow({
  player,
  isCurrent,
  measure,
  index,
  start,
}: {
  player: LeaderboardPlayer;
  isCurrent: boolean;
  measure: (node: HTMLTableRowElement | null) => void;
  index: number;
  start: number;
}) {
  return (
    <TableRow
      ref={measure}
      data-index={index}
      aria-current={isCurrent ? "true" : undefined}
      className={cn(
        "absolute left-0 grid w-full items-center bg-card",
        leaderboardGrid,
        isCurrent && "bg-primary/10 hover:bg-primary/15",
      )}
      style={{ transform: `translateY(${start}px)` }}
    >
      <TableCell className="flex min-w-0 items-center px-2 py-3 sm:px-3">
        <RankBadge rank={player.rank} />
      </TableCell>
      <TableCell className="min-w-0 px-2 py-3 sm:px-3">
        <PlayerIdentity player={player} isCurrent={isCurrent} />
      </TableCell>
      <TableCell className="px-2 py-3 text-right font-semibold tabular-nums sm:px-3">
        {formatNumber(player.wins)}
      </TableCell>
      <TableCell className="px-2 py-3 text-right tabular-nums text-muted-foreground sm:px-3">
        {formatNumber(player.gamesPlayed)}
      </TableCell>
      <TableCell className="hidden px-3 py-3 text-right tabular-nums text-muted-foreground sm:block">
        {formatPlaytime(player.totalPlaytimeSeconds)}
      </TableCell>
      <TableCell className="hidden px-3 py-3 text-right tabular-nums text-muted-foreground lg:block">
        {formatNumber(player.roundsWon)}
      </TableCell>
      <TableCell className="hidden px-3 py-3 text-right tabular-nums text-muted-foreground lg:block">
        {formatNumber(player.pointsInflicted)}
      </TableCell>
    </TableRow>
  );
}

function PlacementRow({ player }: { player: LeaderboardPlayer }) {
  return (
    <div className="border-t bg-muted/60 p-2 sm:p-3">
      <div className="mb-1 px-2 text-[0.68rem] font-medium tracking-[0.14em] text-muted-foreground uppercase">
        {m.your_placement()}
      </div>
      <div className={cn("grid items-center", leaderboardGrid)}>
        <div className="px-2 sm:px-3">
          <RankBadge rank={player.rank} />
        </div>
        <div className="min-w-0 px-2 sm:px-3">
          <PlayerIdentity player={player} isCurrent />
        </div>
        <div className="px-2 text-right font-semibold tabular-nums sm:px-3">
          {formatNumber(player.wins)}
        </div>
        <div className="px-2 text-right tabular-nums text-muted-foreground sm:px-3">
          {formatNumber(player.gamesPlayed)}
        </div>
        <div className="hidden px-3 text-right tabular-nums text-muted-foreground sm:block">
          {formatPlaytime(player.totalPlaytimeSeconds)}
        </div>
        <div className="hidden px-3 text-right tabular-nums text-muted-foreground lg:block">
          {formatNumber(player.roundsWon)}
        </div>
        <div className="hidden px-3 text-right tabular-nums text-muted-foreground lg:block">
          {formatNumber(player.pointsInflicted)}
        </div>
      </div>
    </div>
  );
}

export function LeaderboardPage({ playerId }: { playerId: string }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const query = useInfiniteQuery(leaderboardInfiniteOptions(playerId));
  const players = useMemo(
    () => query.data?.pages.flatMap((page) => page.players) ?? [],
    [query.data],
  );
  const placement = query.data?.pages[0]?.placement ?? null;
  // TanStack Virtual returns intentionally mutable functions.
  // react-doctor-disable-next-line react-hooks-js/incompatible-library
  const virtualizer = useVirtualizer({
    count: players.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 57,
    getItemKey: (index) => players[index]?.playerId ?? index,
    overscan: 10,
  });
  const virtualRows = virtualizer.getVirtualItems();
  const lastVirtualRow = virtualRows.at(-1);

  useEffect(() => {
    if (
      lastVirtualRow &&
      lastVirtualRow.index >= players.length - 8 &&
      query.hasNextPage &&
      !query.isFetchingNextPage &&
      !query.isFetchNextPageError
    ) {
      void query.fetchNextPage();
    }
  }, [
    lastVirtualRow,
    players.length,
    query.fetchNextPage,
    query.hasNextPage,
    query.isFetchNextPageError,
    query.isFetchingNextPage,
  ]);

  return (
    <section className="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col gap-4">
      <header className="flex items-end justify-between gap-4 px-2 pt-2 md:px-0">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex items-center gap-2 text-primary">
            <HugeiconsIcon icon={ChampionIcon} aria-hidden="true" />
            <P size="sm" className="font-medium tracking-[0.16em] uppercase">
              {m.all_time_rankings()}
            </P>
          </div>
          <H1>{m.leaderboard()}</H1>
          <P className="max-w-2xl text-muted-foreground">{m.leaderboard_description()}</P>
        </div>
        <Badge variant="outline" className="hidden shrink-0 sm:inline-flex">
          {m.ranked_by_wins()}
        </Badge>
      </header>

      <Card className="min-h-0 flex-1 gap-0 py-0">
        <CardHeader className="border-b py-4">
          <CardTitle>{m.global_standings()}</CardTitle>
          <CardDescription>{m.leaderboard_scroll_hint()}</CardDescription>
        </CardHeader>

        {query.isPending ? (
          <CardContent className="flex min-h-80 flex-col items-center justify-center gap-3">
            <Spinner className="size-5" />
            <P size="sm" className="text-muted-foreground">
              {m.loading_leaderboard()}
            </P>
          </CardContent>
        ) : query.isError && players.length === 0 ? (
          <CardContent className="flex min-h-80 items-center justify-center p-6">
            <Empty className="max-w-md border-0 p-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>{m.leaderboard_error_title()}</EmptyTitle>
                <EmptyDescription>{m.leaderboard_error_description()}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button size="sm" onClick={() => void query.refetch()}>
                  <HugeiconsIcon icon={Refresh01Icon} data-icon="inline-start" />
                  {m.try_again()}
                </Button>
              </EmptyContent>
            </Empty>
          </CardContent>
        ) : players.length === 0 ? (
          <Empty className="min-h-80 border-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HugeiconsIcon icon={RankingIcon} />
              </EmptyMedia>
              <EmptyTitle>{m.leaderboard_empty_title()}</EmptyTitle>
              <EmptyDescription>{m.leaderboard_empty_description()}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <>
            <div
              ref={scrollRef}
              className="h-[min(62vh,42rem)] min-h-80 overflow-auto overscroll-contain"
            >
              <table className="grid w-full text-sm" aria-label={m.leaderboard()}>
                <LeaderboardHeader />
                <TableBody
                  className="relative grid"
                  style={{ height: `${virtualizer.getTotalSize()}px` }}
                >
                  {virtualRows.map((virtualRow) => {
                    const player = players[virtualRow.index];
                    if (!player) return null;
                    return (
                      <LeaderboardTableRow
                        key={player.playerId}
                        player={player}
                        isCurrent={player.playerId === playerId}
                        measure={virtualizer.measureElement}
                        index={virtualRow.index}
                        start={virtualRow.start}
                      />
                    );
                  })}
                </TableBody>
              </table>
            </div>

            <div className="flex min-h-10 items-center justify-center border-t px-4 py-2 text-xs text-muted-foreground">
              {query.isFetchingNextPage ? (
                <span className="flex items-center gap-2">
                  <Spinner className="size-3" />
                  {m.loading_more_players()}
                </span>
              ) : query.isFetchNextPageError ? (
                <Button variant="ghost" size="xs" onClick={() => void query.fetchNextPage()}>
                  <HugeiconsIcon icon={Refresh01Icon} data-icon="inline-start" />
                  {m.retry_loading_players()}
                </Button>
              ) : query.hasNextPage ? (
                m.scroll_for_more_players()
              ) : (
                m.all_players_loaded()
              )}
            </div>

            {placement ? <PlacementRow player={placement} /> : null}
          </>
        )}
      </Card>
    </section>
  );
}

import { Alert02Icon, ChampionIcon, RankingIcon, Refresh01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useMemo, useRef, useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import { Spinner } from "#/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";
import { Tabs, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { ToggleGroup, ToggleGroupItem } from "#/components/ui/toggle-group";
import { H1, P } from "#/components/typography";
import {
  DEFAULT_LEADERBOARD_METRIC,
  DEFAULT_LEADERBOARD_SCOPE,
  leaderboardInfiniteOptions,
  leaderboardMetricSchema,
  leaderboardScopeSchema,
  type LeaderboardMetric,
  type LeaderboardPlayer,
  type LeaderboardScope,
} from "#/lib/leaderboard";
import { cn, getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { getLocale, type Locale } from "#/paraglide/runtime.js";

const NUMBER_FORMATTERS: Record<Locale, Intl.NumberFormat> = {
  en: new Intl.NumberFormat("en"),
  lv: new Intl.NumberFormat("lv"),
};

const LEADERBOARD_METRICS: LeaderboardMetric[] = ["wins", "games", "playtime", "rounds", "points"];

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

function getMetricLabel(metric: LeaderboardMetric) {
  switch (metric) {
    case "wins":
      return m.wins();
    case "games":
      return m.games_label();
    case "playtime":
      return m.total_playtime();
    case "rounds":
      return m.round_wins_label();
    case "points":
      return m.points_inflicted();
  }
}

function formatScore(metric: LeaderboardMetric, score: number) {
  return metric === "playtime" ? formatPlaytime(score) : formatNumber(score);
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

function PlayerIdentity({ player }: { player: LeaderboardPlayer }) {
  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <Avatar size="sm" className="hidden shrink-0 sm:flex">
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
    </div>
  );
}

function LeaderboardHeader({ metric }: { metric: LeaderboardMetric }) {
  return (
    <TableHeader className="sticky top-0 z-20 bg-card shadow-sm">
      <TableRow className="hover:bg-transparent">
        <TableHead>
          <span aria-hidden="true">#</span>
          <span className="sr-only">{m.rank()}</span>
        </TableHead>
        <TableHead>{m.player()}</TableHead>
        <TableHead className="text-right">{getMetricLabel(metric)}</TableHead>
      </TableRow>
    </TableHeader>
  );
}

function LeaderboardTableRow({
  player,
  isCurrent,
  metric,
  measure,
  index,
}: {
  player: LeaderboardPlayer;
  isCurrent: boolean;
  metric: LeaderboardMetric;
  measure: (node: Element | null) => void;
  index: number;
}) {
  return (
    <TableRow
      ref={measure}
      data-index={index}
      aria-current={isCurrent ? "true" : undefined}
      className={cn(isCurrent && "bg-primary/10 hover:bg-primary/15")}
    >
      <TableCell>
        <RankBadge rank={player.rank} />
      </TableCell>
      <TableCell className="max-w-0">
        <PlayerIdentity player={player} />
      </TableCell>
      <TableCell className="text-right font-semibold tabular-nums">
        {formatScore(metric, player.score)}
      </TableCell>
    </TableRow>
  );
}

function PlacementFooter({
  player,
  metric,
}: {
  player: LeaderboardPlayer;
  metric: LeaderboardMetric;
}) {
  return (
    <TableFooter className="sticky bottom-0 z-20 bg-muted shadow-[0_-1px_0_var(--border)]">
      <TableRow aria-label={m.your_placement()}>
        <TableCell>
          <RankBadge rank={player.rank} />
        </TableCell>
        <TableCell className="max-w-0">
          <PlayerIdentity player={player} />
        </TableCell>
        <TableCell className="text-right font-semibold tabular-nums">
          {formatScore(metric, player.score)}
        </TableCell>
      </TableRow>
    </TableFooter>
  );
}

export function LeaderboardPage({ playerId }: { playerId: string }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [metric, setMetric] = useState<LeaderboardMetric>(DEFAULT_LEADERBOARD_METRIC);
  const [scope, setScope] = useState<LeaderboardScope>(DEFAULT_LEADERBOARD_SCOPE);
  // Query refetching is the intended response to changing the selected metric.
  // react-doctor-disable-next-line react-hooks-js/no-event-handler, react-doctor/no-event-handler
  const queryOptions = leaderboardInfiniteOptions(playerId, metric, scope);
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isError,
    isFetchNextPageError,
    isFetchingNextPage,
    isPending,
    refetch,
  } = useInfiniteQuery(queryOptions);
  // The virtualizer consumes this array identity through getItemKey.
  // react-doctor-disable-next-line react-hooks-js/react-compiler-no-manual-memoization, react-doctor/react-compiler-no-manual-memoization
  const players = useMemo(() => data?.pages.flatMap((page) => page.players) ?? [], [data]);
  const placement = data?.pages[0]?.placement ?? null;

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
  const firstVirtualRow = virtualRows[0];
  const lastVirtualRow = virtualRows.at(-1);
  const topPadding = firstVirtualRow?.start ?? 0;
  const bottomPadding = lastVirtualRow
    ? Math.max(0, virtualizer.getTotalSize() - lastVirtualRow.end)
    : 0;
  const isPlacementVisible =
    placement !== null &&
    virtualRows.some((virtualRow) => players[virtualRow.index]?.playerId === placement.playerId);

  useEffect(() => {
    if (
      lastVirtualRow &&
      lastVirtualRow.index >= players.length - 8 &&
      hasNextPage &&
      !isFetchingNextPage &&
      !isFetchNextPageError
    ) {
      void fetchNextPage();
    }
  }, [
    lastVirtualRow,
    players.length,
    fetchNextPage,
    hasNextPage,
    isFetchNextPageError,
    isFetchingNextPage,
  ]);

  function handleMetricChange(value: string) {
    const parsed = leaderboardMetricSchema.safeParse(value);
    if (!parsed.success) return;
    scrollRef.current?.scrollTo({ top: 0 });
    setMetric(parsed.data);
  }

  function handleScopeChange(value: string[]) {
    const parsed = leaderboardScopeSchema.safeParse(value[0]);
    if (!parsed.success) return;
    scrollRef.current?.scrollTo({ top: 0 });
    setScope(parsed.data);
  }

  return (
    <section className="mx-auto w-full max-w-6xl space-y-4">
      <header className="px-2 pt-2 md:px-0">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-primary">
              <HugeiconsIcon icon={ChampionIcon} aria-hidden="true" />
              <P size="sm" className="font-medium tracking-[0.16em] uppercase">
                {m.all_time_rankings()}
              </P>
            </div>
            <H1 className="mt-1">{m.leaderboard()}</H1>
          </div>
          <ToggleGroup
            value={[scope]}
            onValueChange={handleScopeChange}
            variant="outline"
            spacing={0}
            size="sm"
            aria-label={m.leaderboard_scope()}
          >
            <ToggleGroupItem value="friends">{m.friends()}</ToggleGroupItem>
            <ToggleGroupItem value="global">{m.global()}</ToggleGroupItem>
          </ToggleGroup>
        </div>
      </header>

      <Card className="gap-0 overflow-hidden py-0">
        <Tabs value={metric} onValueChange={handleMetricChange} className="gap-0">
          <div className="overflow-x-auto border-b p-3">
            <TabsList aria-label={m.leaderboard_statistic()}>
              {LEADERBOARD_METRICS.map((item) => (
                <TabsTrigger key={item} value={item}>
                  {getMetricLabel(item)}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>

          {isPending ? (
            <CardContent className="flex items-center justify-center gap-3 py-14">
              <Spinner className="size-5" />
              <P size="sm" className="text-muted-foreground">
                {m.loading_leaderboard()}
              </P>
            </CardContent>
          ) : isError && players.length === 0 ? (
            <CardContent className="flex items-center justify-center p-6 py-12">
              <Empty className="max-w-md border-0 p-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
                  </EmptyMedia>
                  <EmptyTitle>{m.leaderboard_error_title()}</EmptyTitle>
                  <EmptyDescription>{m.leaderboard_error_description()}</EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button size="sm" onClick={() => void refetch()}>
                    <HugeiconsIcon icon={Refresh01Icon} data-icon="inline-start" />
                    {m.try_again()}
                  </Button>
                </EmptyContent>
              </Empty>
            </CardContent>
          ) : players.length === 0 ? (
            <Empty className="border-0 py-12">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <HugeiconsIcon icon={RankingIcon} />
                </EmptyMedia>
                <EmptyTitle>
                  {scope === "friends"
                    ? m.friends_leaderboard_empty_title()
                    : m.leaderboard_empty_title()}
                </EmptyTitle>
                <EmptyDescription>
                  {scope === "friends"
                    ? m.friends_leaderboard_empty_description()
                    : m.leaderboard_empty_description()}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <>
              <Table
                containerRef={scrollRef}
                className="min-w-[32rem]"
                containerClassName="max-h-[min(62dvh,42rem)] touch-auto overflow-auto overscroll-contain"
                aria-label={`${m.leaderboard()}: ${scope === "friends" ? m.friends() : m.global()}, ${getMetricLabel(metric)}`}
              >
                <colgroup>
                  <col className="w-20" />
                  <col />
                  <col />
                </colgroup>
                <LeaderboardHeader metric={metric} />
                <TableBody>
                  {topPadding > 0 ? (
                    <TableRow
                      aria-hidden="true"
                      className="pointer-events-none border-0 hover:bg-transparent"
                    >
                      <TableCell colSpan={3} className="p-0" style={{ height: topPadding }} />
                    </TableRow>
                  ) : null}
                  {virtualRows.map((virtualRow) => {
                    const player = players[virtualRow.index];
                    if (!player) return null;
                    return (
                      <LeaderboardTableRow
                        key={player.playerId}
                        player={player}
                        isCurrent={player.playerId === playerId}
                        metric={metric}
                        measure={virtualizer.measureElement}
                        index={virtualRow.index}
                      />
                    );
                  })}
                  {bottomPadding > 0 ? (
                    <TableRow
                      aria-hidden="true"
                      className="pointer-events-none border-0 hover:bg-transparent"
                    >
                      <TableCell colSpan={3} className="p-0" style={{ height: bottomPadding }} />
                    </TableRow>
                  ) : null}
                </TableBody>
                {placement && !isPlacementVisible ? (
                  <PlacementFooter player={placement} metric={metric} />
                ) : null}
              </Table>

              {isFetchingNextPage || isFetchNextPageError || hasNextPage ? (
                <div className="flex min-h-10 items-center justify-center border-t px-4 py-2 text-xs text-muted-foreground">
                  {isFetchingNextPage ? (
                    <span className="flex items-center gap-2">
                      <Spinner className="size-3" />
                      {m.loading_more_players()}
                    </span>
                  ) : isFetchNextPageError ? (
                    <Button variant="ghost" size="xs" onClick={() => void fetchNextPage()}>
                      <HugeiconsIcon icon={Refresh01Icon} data-icon="inline-start" />
                      {m.retry_loading_players()}
                    </Button>
                  ) : (
                    m.scroll_for_more_players()
                  )}
                </div>
              ) : null}
            </>
          )}
        </Tabs>
      </Card>
    </section>
  );
}

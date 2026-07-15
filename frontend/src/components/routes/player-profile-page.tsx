import {
  ArrowLeft01Icon,
  ArrowLeft02Icon,
  ArrowRight01Icon,
  Share08Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import { Separator } from "#/components/ui/separator";
import { Spinner } from "#/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";
import { Caption, H1, H2, P } from "#/components/typography";
import {
  getPlayerGameHistory,
  type PlayerGameHistory,
  type PlayerProfile,
} from "#/lib/player-profile";
import { getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { getLocale } from "#/paraglide/runtime.js";

function ratio(numerator: number, denominator: number) {
  return denominator > 0 ? numerator / denominator : null;
}

function formatPercent(value: number | null) {
  return value === null
    ? "—"
    : new Intl.NumberFormat(getLocale(), { style: "percent", maximumFractionDigits: 0 }).format(
        value,
      );
}

function formatDecimal(value: number | null) {
  return value === null
    ? "—"
    : new Intl.NumberFormat(getLocale(), { maximumFractionDigits: 1 }).format(value);
}

function formatPlaytime(totalSeconds: number) {
  const totalMinutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  if (hours > 0) return m.playtime_hours_minutes({ hours, minutes });
  return m.playtime_minutes({ minutes: totalMinutes });
}

function StatCard({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <Card size="sm" className="h-full">
      <CardHeader className="gap-2">
        <CardDescription className="min-h-4 text-xs/4 font-medium tracking-[0.08em] text-pretty uppercase">
          {label}
        </CardDescription>
        <H2 className="font-medium tracking-tight tabular-nums" data-slot="card-title">
          {value}
        </H2>
      </CardHeader>
      <CardContent className="mt-auto">
        <Caption className="text-pretty">{note}</Caption>
      </CardContent>
    </Card>
  );
}

function formatCompletedAt(value: string) {
  return new Intl.DateTimeFormat(getLocale(), {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function GameHistory({
  playerId,
  initialHistory,
}: {
  playerId: string;
  initialHistory: PlayerGameHistory;
}) {
  const [page, setPage] = useState(1);
  const { data, isError, isFetching } = useQuery({
    queryKey: ["player-game-history", playerId, page],
    queryFn: () => getPlayerGameHistory({ data: { playerId, page, pageSize: 10 } }),
    initialData: page === 1 ? initialHistory : undefined,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
  const history = data ?? initialHistory;

  return (
    <Card>
      <CardHeader className="border-b">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <CardTitle>{m.game_history()}</CardTitle>
            <CardDescription>{m.game_history_description()}</CardDescription>
          </div>
          <Badge variant="outline" className="mt-0.5 tabular-nums">
            {m.ranked_games({ count: history.totalItems })}
          </Badge>
        </div>
      </CardHeader>

      {isError ? (
        <CardContent>
          <P size="sm" className="text-destructive">
            {m.game_history_error()}
          </P>
        </CardContent>
      ) : (
        <Table aria-label={m.game_history()} className={isFetching ? "opacity-60" : undefined}>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>{m.played_at()}</TableHead>
              <TableHead>{m.result()}</TableHead>
              <TableHead className="text-right">{m.place()}</TableHead>
              <TableHead className="text-right">{m.players()}</TableHead>
              <TableHead className="text-right">{m.rounds()}</TableHead>
              <TableHead className="text-right">{m.score()}</TableHead>
              <TableHead className="text-right">{m.duration()}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {history.games.map((game) => (
              <TableRow key={game.id}>
                <TableCell className="font-medium">{formatCompletedAt(game.completedAt)}</TableCell>
                <TableCell>
                  <Badge
                    variant={game.won ? "default" : game.forfeited ? "destructive" : "secondary"}
                  >
                    {game.won ? m.victory() : game.forfeited ? m.forfeit() : m.finished()}
                  </Badge>
                </TableCell>
                <TableCell className="text-right font-medium tabular-nums">
                  {game.placement}
                </TableCell>
                <TableCell className="text-right tabular-nums">{game.playerCount}</TableCell>
                <TableCell className="text-right tabular-nums">
                  <span className="font-medium">{game.roundsWon}</span>
                  <span className="text-muted-foreground"> / {game.roundsPlayed}</span>
                </TableCell>
                <TableCell className="text-right tabular-nums">{game.totalPoints}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatPlaytime(game.playtimeSeconds)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CardFooter className="justify-between gap-3 border-t">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1 || isFetching}
          onClick={() => setPage((current) => Math.max(1, current - 1))}
        >
          <HugeiconsIcon icon={ArrowLeft02Icon} data-icon="inline-start" />
          {m.previous()}
        </Button>
        <div className="flex items-center gap-2 text-xs text-muted-foreground tabular-nums">
          {isFetching ? <Spinner className="size-3" /> : null}
          {m.page_of({ page: history.page, total: Math.max(history.totalPages, 1) })}
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= history.totalPages || isFetching}
          onClick={() => setPage((current) => current + 1)}
        >
          {m.next()}
          <HugeiconsIcon icon={ArrowRight01Icon} data-icon="inline-end" />
        </Button>
      </CardFooter>
    </Card>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <P size="sm" className="text-muted-foreground">
        {label}
      </P>
      <P size="sm" className="font-medium tabular-nums">
        {value}
      </P>
    </div>
  );
}

export function PlayerProfilePage({
  profile,
  initialHistory,
  isOwnProfile,
}: {
  profile: PlayerProfile;
  initialHistory: PlayerGameHistory;
  isOwnProfile: boolean;
}) {
  const hasGames = profile.gamesPlayed > 0;
  const compositions = profile.compositionsCreated;

  async function shareProfile() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      toast.success(m.profile_link_copied());
    } catch {
      toast.error(m.profile_link_copy_failed());
    }
  }

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <Button render={<Link to="/" />} nativeButton={false} variant="ghost" size="sm">
          <HugeiconsIcon icon={ArrowLeft01Icon} data-icon="inline-start" />
          {m.back_to_game()}
        </Button>
        {isOwnProfile ? (
          <Button variant="ghost" size="sm" onClick={() => void shareProfile()}>
            <HugeiconsIcon icon={Share08Icon} strokeWidth={2} data-icon="inline-start" />
            {m.share_profile()}
          </Button>
        ) : null}
      </div>

      <Card className="overflow-hidden">
        <CardHeader>
          <div className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] items-center gap-4">
            <Avatar className="size-16 sm:size-20">
              {profile.imageUrl ? <AvatarImage src={profile.imageUrl} alt={profile.name} /> : null}
              <AvatarFallback>
                <P
                  size="lg"
                  className="font-heading text-lg/6 font-semibold tracking-tight md:text-xl/7"
                >
                  {getUserInitials(profile.name)}
                </P>
              </AvatarFallback>
            </Avatar>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <Caption
                className="font-medium tracking-[0.18em] uppercase"
                data-slot="card-description"
              >
                {m.player_profile()}
              </Caption>
              <H1 className="truncate">{profile.name}</H1>
              <Badge className="mt-1 w-fit" variant={hasGames ? "secondary" : "outline"}>
                {hasGames ? m.ranked_games({ count: profile.gamesPlayed }) : m.unranked()}
              </Badge>
            </div>
          </div>
        </CardHeader>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <StatCard
          label={m.games_played()}
          value={String(profile.gamesPlayed)}
          note={m.ranked_finishes()}
        />
        <StatCard
          label={m.total_playtime()}
          value={formatPlaytime(profile.totalPlaytimeSeconds)}
          note={m.completed_ranked_games()}
        />
        <StatCard
          label={m.win_rate()}
          value={formatPercent(ratio(profile.gamesWon, profile.gamesPlayed))}
          note={m.game_wins({ count: profile.gamesWon })}
        />
        <StatCard
          label={m.average_finish()}
          value={formatDecimal(ratio(profile.totalPlacement, profile.gamesPlayed))}
          note={m.lower_better()}
        />
        <StatCard
          label={m.round_win_rate()}
          value={formatPercent(ratio(profile.roundsWon, profile.roundsPlayed))}
          note={m.round_wins({ count: profile.roundsWon })}
        />
      </div>

      {!hasGames ? (
        <Card>
          <CardHeader>
            <CardTitle>{m.no_ranked_history()}</CardTitle>
            <CardDescription>{m.stats_after_first_game()}</CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>{m.round_craft()}</CardTitle>
              <CardDescription>{m.round_craft_description()}</CardDescription>
            </CardHeader>
            <CardContent>
              <DetailRow label={m.rounds_played()} value={String(profile.roundsPlayed)} />
              <Separator />
              <DetailRow
                label={m.round_win_rate()}
                value={formatPercent(ratio(profile.roundsWon, profile.roundsPlayed))}
              />
              <Separator />
              <DetailRow label={m.compositions_created()} value={String(compositions)} />
              <Separator />
              <DetailRow
                label={m.runs_sets()}
                value={`${profile.runsCreated} / ${profile.setsCreated}`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{m.competitive_record()}</CardTitle>
              <CardDescription>{m.competitive_record_description()}</CardDescription>
            </CardHeader>
            <CardContent>
              <DetailRow
                label={m.current_win_streak()}
                value={String(profile.currentGameWinStreak)}
              />
              <Separator />
              <DetailRow label={m.best_win_streak()} value={String(profile.longestGameWinStreak)} />
              <Separator />
              <DetailRow label={m.points_inflicted()} value={String(profile.pointsInflicted)} />
              <Separator />
              <DetailRow label={m.penalty_points()} value={String(profile.penaltyPoints)} />
            </CardContent>
          </Card>
        </div>
      )}

      {hasGames ? <GameHistory playerId={profile.id} initialHistory={initialHistory} /> : null}
    </section>
  );
}

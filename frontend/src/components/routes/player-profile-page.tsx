import { ArrowLeft01Icon, Share08Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { toast } from "sonner";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Separator } from "#/components/ui/separator";
import { Caption, H1, H2, P } from "#/components/typography";
import type { PlayerProfile } from "#/lib/player-profile";
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
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <H2 className="font-medium tabular-nums" data-slot="card-title">
          {value}
        </H2>
      </CardHeader>
      <CardContent>
        <Caption>{note}</Caption>
      </CardContent>
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
  isOwnProfile,
}: {
  profile: PlayerProfile;
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
                <P size="lg" className="font-heading text-xl/7 font-semibold tracking-tight">
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
    </section>
  );
}

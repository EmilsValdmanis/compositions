import { ArrowLeft01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Separator } from "#/components/ui/separator";
import type { PlayerProfile } from "#/lib/player-profile";
import { getUserInitials } from "#/lib/utils";

function ratio(numerator: number, denominator: number) {
  return denominator > 0 ? numerator / denominator : null;
}

function formatPercent(value: number | null) {
  return value === null
    ? "—"
    : new Intl.NumberFormat(undefined, { style: "percent", maximumFractionDigits: 0 }).format(
        value,
      );
}

function formatDecimal(value: number | null) {
  return value === null
    ? "—"
    : new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(value);
}

function StatCard({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-3xl tracking-tight tabular-nums">{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">{note}</p>
      </CardContent>
    </Card>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{value}</span>
    </div>
  );
}

export function PlayerProfilePage({ profile }: { profile: PlayerProfile }) {
  const hasGames = profile.gamesPlayed > 0;
  const compositions = profile.compositionsCreated;

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-4">
      <div>
        <Button render={<Link to="/" />} nativeButton={false} variant="ghost" size="sm">
          <HugeiconsIcon icon={ArrowLeft01Icon} data-icon="inline-start" />
          Back to game
        </Button>
      </div>

      <Card className="relative overflow-hidden">
        <div className="pointer-events-none absolute -top-20 -right-20 size-64 rounded-full bg-primary/10 blur-3xl" />
        <CardHeader className="relative">
          <div className="flex flex-wrap items-center gap-4">
            <Avatar className="size-20 ring-4 ring-background shadow-md">
              {profile.imageUrl ? <AvatarImage src={profile.imageUrl} alt={profile.name} /> : null}
              <AvatarFallback className="text-xl">{getUserInitials(profile.name)}</AvatarFallback>
            </Avatar>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <CardDescription className="uppercase tracking-[0.18em]">
                Player profile
              </CardDescription>
              <CardTitle className="truncate text-3xl tracking-tight md:text-4xl">
                {profile.name}
              </CardTitle>
            </div>
            <Badge variant={hasGames ? "secondary" : "outline"}>
              {hasGames ? `${profile.gamesPlayed} ranked games` : "Unranked"}
            </Badge>
          </div>
        </CardHeader>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Games played" value={String(profile.gamesPlayed)} note="Ranked finishes" />
        <StatCard
          label="Win rate"
          value={formatPercent(ratio(profile.gamesWon, profile.gamesPlayed))}
          note={`${profile.gamesWon} game wins`}
        />
        <StatCard
          label="Average finish"
          value={formatDecimal(ratio(profile.totalPlacement, profile.gamesPlayed))}
          note="Lower is better"
        />
        <StatCard
          label="Round win rate"
          value={formatPercent(ratio(profile.roundsWon, profile.roundsPlayed))}
          note={`${profile.roundsWon} round wins`}
        />
      </div>

      {!hasGames ? (
        <Card>
          <CardHeader>
            <CardTitle>No ranked history yet</CardTitle>
            <CardDescription>
              Statistics will appear here after this player completes their first ranked game.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Round craft</CardTitle>
              <CardDescription>How this player builds and closes rounds.</CardDescription>
            </CardHeader>
            <CardContent>
              <DetailRow label="Rounds played" value={String(profile.roundsPlayed)} />
              <Separator />
              <DetailRow
                label="Round win rate"
                value={formatPercent(ratio(profile.roundsWon, profile.roundsPlayed))}
              />
              <Separator />
              <DetailRow label="Compositions created" value={String(compositions)} />
              <Separator />
              <DetailRow
                label="Runs / sets"
                value={`${profile.runsCreated} / ${profile.setsCreated}`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Competitive record</CardTitle>
              <CardDescription>Streaks and scoring across ranked games.</CardDescription>
            </CardHeader>
            <CardContent>
              <DetailRow label="Current win streak" value={String(profile.currentGameWinStreak)} />
              <Separator />
              <DetailRow label="Best win streak" value={String(profile.longestGameWinStreak)} />
              <Separator />
              <DetailRow label="Points inflicted" value={String(profile.pointsInflicted)} />
              <Separator />
              <DetailRow label="Penalty points taken" value={String(profile.penaltyPoints)} />
            </CardContent>
          </Card>
        </div>
      )}
    </section>
  );
}

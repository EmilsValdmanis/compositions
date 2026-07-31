import { Alert02Icon, Calendar03Icon, RefreshIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { enUS, lv } from "date-fns/locale";
import { useState, type ReactNode } from "react";
import type { DateRange } from "react-day-picker";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  XAxis,
  YAxis,
} from "recharts";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Calendar } from "#/components/ui/calendar";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "#/components/ui/chart";
import { Popover, PopoverContent, PopoverTrigger } from "#/components/ui/popover";
import { Skeleton } from "#/components/ui/skeleton";
import { H2 } from "#/components/typography";
import {
  adminAnalyticsOptions,
  type AdminAnalytics,
  type AdminAnalyticsRange,
  type AdminAnalyticsTotals,
} from "#/lib/admin-analytics";
import { currentAnalyticsDate, shiftCalendarDate } from "#/lib/admin-analytics-date";
import { m } from "#/paraglide/messages.js";
import { getLocale, type Locale } from "#/paraglide/runtime.js";

const NUMBER_FORMATTERS: Record<Locale, Intl.NumberFormat> = {
  en: new Intl.NumberFormat("en"),
  lv: new Intl.NumberFormat("lv"),
};

const PERCENT_FORMATTERS: Record<Locale, Intl.NumberFormat> = {
  en: new Intl.NumberFormat("en", { style: "percent", maximumFractionDigits: 1 }),
  lv: new Intl.NumberFormat("lv", { style: "percent", maximumFractionDigits: 1 }),
};

const AXIS_DATE_FORMATTERS: Record<Locale, Intl.DateTimeFormat> = {
  en: new Intl.DateTimeFormat("en", { month: "short", day: "numeric" }),
  lv: new Intl.DateTimeFormat("lv", { month: "short", day: "numeric" }),
};

const FULL_DATE_FORMATTERS: Record<Locale, Intl.DateTimeFormat> = {
  en: new Intl.DateTimeFormat("en", { dateStyle: "medium" }),
  lv: new Intl.DateTimeFormat("lv", { dateStyle: "medium" }),
};

const DATE_FNS_LOCALES = { en: enUS, lv } as const;

function defaultDateRange(): AdminAnalyticsRange {
  const to = currentAnalyticsDate();
  return { from: shiftCalendarDate(to, -29), to };
}

function parseCalendarDate(value: string) {
  return parseISO(value);
}

function formatAxisDate(value: string) {
  return AXIS_DATE_FORMATTERS[getLocale()].format(parseCalendarDate(value));
}

function formatFullDate(value: string) {
  return FULL_DATE_FORMATTERS[getLocale()].format(parseCalendarDate(value));
}

function formatDateRange(value: AdminAnalyticsRange) {
  return `${formatFullDate(value.from)} – ${formatFullDate(value.to)}`;
}

function DateRangePicker({
  value,
  onChange,
}: {
  value: AdminAnalyticsRange;
  onChange: (value: AdminAnalyticsRange) => void;
}) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<DateRange | undefined>({
    from: parseCalendarDate(value.from),
    to: parseCalendarDate(value.to),
  });
  const todayValue = currentAnalyticsDate();
  const today = parseCalendarDate(todayValue);
  const earliestDate = parseCalendarDate(shiftCalendarDate(todayValue, -365));

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (nextOpen) {
          setDraft({ from: parseCalendarDate(value.from), to: parseCalendarDate(value.to) });
        }
      }}
    >
      <PopoverTrigger
        render={<Button variant="outline" className="w-full justify-start sm:w-auto" />}
      >
        <HugeiconsIcon icon={Calendar03Icon} data-icon="inline-start" />
        <span className="truncate">{formatDateRange(value)}</span>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-auto p-0">
        <Calendar
          mode="range"
          today={today}
          selected={draft}
          onSelect={(nextRange) => {
            setDraft(nextRange);
            if (nextRange?.from && nextRange.to) {
              onChange({
                from: format(nextRange.from, "yyyy-MM-dd"),
                to: format(nextRange.to, "yyyy-MM-dd"),
              });
              setOpen(false);
            }
          }}
          locale={DATE_FNS_LOCALES[getLocale()]}
          disabled={{ before: earliestDate, after: today }}
          defaultMonth={draft?.from}
          autoFocus
        />
      </PopoverContent>
    </Popover>
  );
}

function relativeDelta(current: number, previous: number) {
  if (previous === 0) return current === 0 ? null : Number.POSITIVE_INFINITY;
  return (current - previous) / previous;
}

function DeltaBadge({ current, previous }: { current: number; previous: number }) {
  const delta = relativeDelta(current, previous);
  if (delta === null) return <Badge variant="outline">—</Badge>;
  if (!Number.isFinite(delta)) return <Badge variant="secondary">{m.admin_analytics_new()}</Badge>;
  const label = PERCENT_FORMATTERS[getLocale()].format(Math.abs(delta));
  return (
    <Badge variant={delta < 0 ? "destructive" : delta > 0 ? "secondary" : "outline"}>
      {delta > 0 ? "+" : delta < 0 ? "−" : ""}
      {label}
    </Badge>
  );
}

function PointDeltaBadge({ current, previous }: { current: number; previous: number }) {
  const points = (current - previous) * 100;
  const label = NUMBER_FORMATTERS[getLocale()].format(Math.abs(points));
  return (
    <Badge variant={points < 0 ? "destructive" : points > 0 ? "secondary" : "outline"}>
      {points > 0 ? "+" : points < 0 ? "−" : ""}
      {label} {m.admin_analytics_percentage_points_short()}
    </Badge>
  );
}

function formatPlaytime(seconds: number) {
  const minutes = Math.round(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return hours > 0
    ? m.playtime_hours_minutes({ hours, minutes: remainingMinutes })
    : m.playtime_minutes({ minutes });
}

function SummaryCard({
  label,
  value,
  current,
  previous,
  points = false,
}: {
  label: string;
  value: string;
  current: number;
  previous: number;
  points?: boolean;
}) {
  return (
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle>
          <H2 className="tabular-nums">{value}</H2>
        </CardTitle>
        <div className="pt-1">
          {points ? (
            <PointDeltaBadge current={current} previous={previous} />
          ) : (
            <DeltaBadge current={current} previous={previous} />
          )}
        </div>
      </CardHeader>
    </Card>
  );
}

function SummaryGrid({
  current,
  previous,
}: {
  current: AdminAnalyticsTotals;
  previous: AdminAnalyticsTotals;
}) {
  const numberFormatter = NUMBER_FORMATTERS[getLocale()];
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <SummaryCard
        label={m.games_played()}
        value={numberFormatter.format(current.games)}
        current={current.games}
        previous={previous.games}
      />
      <SummaryCard
        label={m.admin_analytics_active_players()}
        value={numberFormatter.format(current.activePlayers)}
        current={current.activePlayers}
        previous={previous.activePlayers}
      />
      <SummaryCard
        label={m.total_playtime()}
        value={formatPlaytime(current.activePlaytimeSeconds)}
        current={current.activePlaytimeSeconds}
        previous={previous.activePlaytimeSeconds}
      />
      <SummaryCard
        label={m.admin_analytics_healthy_finish_rate()}
        value={PERCENT_FORMATTERS[getLocale()].format(current.healthyFinishRate)}
        current={current.healthyFinishRate}
        previous={previous.healthyFinishRate}
        points
      />
    </div>
  );
}

function AnalyticsChartCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <Card size="sm" className="min-w-0">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0">{children}</CardContent>
    </Card>
  );
}

function ChartXAxis() {
  return (
    <XAxis
      dataKey="date"
      tickLine={false}
      axisLine={false}
      tickMargin={8}
      minTickGap={28}
      tickFormatter={formatAxisDate}
    />
  );
}

function ChartYAxis() {
  return <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={28} />;
}

function AnalyticsCharts({ analytics }: { analytics: AdminAnalytics }) {
  const usageConfig = {
    games: { label: m.games_label(), color: "var(--chart-1)" },
    activePlayers: { label: m.admin_analytics_active_players(), color: "var(--chart-3)" },
  } satisfies ChartConfig;
  const playersConfig = {
    newPlayers: { label: m.admin_analytics_new_players(), color: "var(--chart-2)" },
    returningPlayers: { label: m.admin_analytics_returning_players(), color: "var(--chart-4)" },
  } satisfies ChartConfig;
  const outcomeConfig = {
    completed: { label: m.admin_analytics_completed(), color: "var(--chart-1)" },
    forfeit: { label: m.forfeit(), color: "var(--chart-2)" },
    mutualEnd: { label: m.admin_analytics_mutual_end(), color: "var(--chart-3)" },
    technicalAbort: { label: m.admin_analytics_technical_abort(), color: "var(--chart-4)" },
    abandoned: { label: m.admin_analytics_abandoned(), color: "var(--chart-5)" },
    inProgress: { label: m.admin_analytics_in_progress(), color: "var(--muted-foreground)" },
  } satisfies ChartConfig;
  const bugConfig = {
    bugReports: { label: m.admin_analytics_reports_created(), color: "var(--chart-2)" },
    bugsResolved: { label: m.admin_analytics_reports_resolved(), color: "var(--chart-4)" },
  } satisfies ChartConfig;
  const medianResolution =
    analytics.current.bugsResolved > 0
      ? formatPlaytime(analytics.current.medianBugResolutionSeconds)
      : "—";
  const tooltipContent = (
    <ChartTooltipContent
      labelFormatter={(_, payload) => {
        const point = payload[0]?.payload as { date?: string } | undefined;
        return point?.date ? formatFullDate(point.date) : "";
      }}
    />
  );

  return (
    <div className="grid min-w-0 gap-4 lg:grid-cols-2">
      <AnalyticsChartCard
        title={m.admin_analytics_usage_trend()}
        description={m.admin_analytics_usage_trend_description()}
      >
        <ChartContainer config={usageConfig} className="h-64 w-full">
          <LineChart accessibilityLayer data={analytics.points} margin={{ left: 0, right: 12 }}>
            <CartesianGrid vertical={false} />
            <ChartXAxis />
            <ChartYAxis />
            <ChartTooltip content={tooltipContent} />
            <ChartLegend content={<ChartLegendContent />} />
            <Line
              dataKey="games"
              type="monotone"
              stroke="var(--color-games)"
              strokeWidth={2}
              dot={false}
            />
            <Line
              dataKey="activePlayers"
              type="monotone"
              stroke="var(--color-activePlayers)"
              strokeWidth={2}
              dot={false}
            />
          </LineChart>
        </ChartContainer>
      </AnalyticsChartCard>

      <AnalyticsChartCard
        title={m.admin_analytics_player_mix()}
        description={m.admin_analytics_player_mix_description()}
      >
        <ChartContainer config={playersConfig} className="h-64 w-full">
          <BarChart accessibilityLayer data={analytics.points} margin={{ left: 0, right: 12 }}>
            <CartesianGrid vertical={false} />
            <ChartXAxis />
            <ChartYAxis />
            <ChartTooltip content={tooltipContent} />
            <ChartLegend content={<ChartLegendContent />} />
            <Bar dataKey="newPlayers" stackId="players" fill="var(--color-newPlayers)" />
            <Bar
              dataKey="returningPlayers"
              stackId="players"
              fill="var(--color-returningPlayers)"
              radius={[4, 4, 0, 0]}
            />
          </BarChart>
        </ChartContainer>
      </AnalyticsChartCard>

      <AnalyticsChartCard
        title={m.admin_analytics_game_outcomes()}
        description={m.admin_analytics_game_outcomes_description()}
      >
        <ChartContainer config={outcomeConfig} className="h-64 w-full">
          <BarChart accessibilityLayer data={analytics.points} margin={{ left: 0, right: 12 }}>
            <CartesianGrid vertical={false} />
            <ChartXAxis />
            <ChartYAxis />
            <ChartTooltip content={tooltipContent} />
            <ChartLegend content={<ChartLegendContent className="flex-wrap gap-x-3 gap-y-1" />} />
            <Bar dataKey="completed" stackId="outcomes" fill="var(--color-completed)" />
            <Bar dataKey="forfeit" stackId="outcomes" fill="var(--color-forfeit)" />
            <Bar dataKey="mutualEnd" stackId="outcomes" fill="var(--color-mutualEnd)" />
            <Bar dataKey="technicalAbort" stackId="outcomes" fill="var(--color-technicalAbort)" />
            <Bar dataKey="abandoned" stackId="outcomes" fill="var(--color-abandoned)" />
            <Bar
              dataKey="inProgress"
              stackId="outcomes"
              fill="var(--color-inProgress)"
              radius={[4, 4, 0, 0]}
            />
          </BarChart>
        </ChartContainer>
      </AnalyticsChartCard>

      <AnalyticsChartCard
        title={m.admin_analytics_bug_activity()}
        description={m.admin_analytics_bug_activity_description({ median: medianResolution })}
      >
        <ChartContainer config={bugConfig} className="h-64 w-full">
          <AreaChart accessibilityLayer data={analytics.points} margin={{ left: 0, right: 12 }}>
            <CartesianGrid vertical={false} />
            <ChartXAxis />
            <ChartYAxis />
            <ChartTooltip content={tooltipContent} />
            <ChartLegend content={<ChartLegendContent />} />
            <Area
              dataKey="bugReports"
              type="monotone"
              fill="var(--color-bugReports)"
              fillOpacity={0.18}
              stroke="var(--color-bugReports)"
              strokeWidth={2}
            />
            <Area
              dataKey="bugsResolved"
              type="monotone"
              fill="var(--color-bugsResolved)"
              fillOpacity={0.18}
              stroke="var(--color-bugsResolved)"
              strokeWidth={2}
            />
          </AreaChart>
        </ChartContainer>
      </AnalyticsChartCard>
    </div>
  );
}

function AnalyticsSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <Card key={index} size="sm">
            <CardHeader>
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-8 w-32" />
              <Skeleton className="h-5 w-16" />
            </CardHeader>
          </Card>
        ))}
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        {Array.from({ length: 4 }, (_, index) => (
          <Card key={index} size="sm">
            <CardHeader>
              <Skeleton className="h-5 w-36" />
              <Skeleton className="h-4 w-56 max-w-full" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-64 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

export function AdminAnalyticsDashboard() {
  const [dateRange, setDateRange] = useState(defaultDateRange);
  const { data, isError, isPending, refetch } = useQuery(adminAnalyticsOptions(dateRange));

  return (
    <Card className="min-w-0">
      <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1.5">
          <CardTitle>{m.admin_analytics_title()}</CardTitle>
          <CardDescription>{m.admin_analytics_description()}</CardDescription>
        </div>
        <DateRangePicker value={dateRange} onChange={setDateRange} />
      </CardHeader>
      <CardContent>
        {isError ? (
          <Alert variant="destructive">
            <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
            <AlertTitle>{m.admin_analytics_error()}</AlertTitle>
            <AlertDescription className="flex flex-col items-start gap-3">
              <span>{m.admin_analytics_error_description()}</span>
              <Button variant="outline" size="sm" onClick={() => void refetch()}>
                <HugeiconsIcon icon={RefreshIcon} data-icon="inline-start" />
                {m.try_again()}
              </Button>
            </AlertDescription>
          </Alert>
        ) : isPending || !data ? (
          <AnalyticsSkeleton />
        ) : (
          <div className="flex min-w-0 flex-col gap-4">
            <SummaryGrid current={data.current} previous={data.previous} />
            <AnalyticsCharts analytics={data} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

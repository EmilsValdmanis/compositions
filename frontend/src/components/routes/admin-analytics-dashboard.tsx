import {
  Alert02Icon,
  ArrowLeftIcon,
  ArrowRightIcon,
  Calendar03Icon,
  RefreshIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import { differenceInCalendarDays, format, parseISO } from "date-fns";
import { enUS, lv } from "date-fns/locale";
import { useRef, useState, type ReactNode } from "react";
import type { DateRange } from "react-day-picker";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  Rectangle,
  XAxis,
  YAxis,
  type BarShapeProps,
} from "recharts";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
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
import { Separator } from "#/components/ui/separator";
import { Skeleton } from "#/components/ui/skeleton";
import { ToggleGroup, ToggleGroupItem } from "#/components/ui/toggle-group";
import { H2 } from "#/components/typography";
import {
  adminAnalyticsOptions,
  type AdminAnalytics,
  type AdminAnalyticsRange,
  type AdminAnalyticsTotals,
} from "#/lib/admin-analytics";
import {
  analyticsPeriodRange,
  currentAnalyticsDate,
  shiftAnalyticsPeriod,
  type AnalyticsPeriod,
} from "#/lib/admin-analytics-date";
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

export function defaultDateRange(): AdminAnalyticsRange {
  return analyticsPeriodRange("month", currentAnalyticsDate());
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

export function DateRangePicker({
  value,
  onChange,
}: {
  value: AdminAnalyticsRange;
  onChange: (value: AdminAnalyticsRange) => void;
}) {
  const [open, setOpen] = useState(false);
  const [period, setPeriod] = useState<AnalyticsPeriod>("month");
  const [draft, setDraft] = useState<DateRange | undefined>(() => ({
    from: parseCalendarDate(value.from),
    to: parseCalendarDate(value.to),
  }));
  const rangeStartRef = useRef<Date | undefined>(undefined);
  const [month, setMonth] = useState(() => parseCalendarDate(value.from));
  const todayValue = currentAnalyticsDate();
  const today = parseCalendarDate(todayValue);

  function applyRange(nextRange: AdminAnalyticsRange, nextPeriod: AnalyticsPeriod) {
    setPeriod(nextPeriod);
    setDraft({
      from: parseCalendarDate(nextRange.from),
      to: parseCalendarDate(nextRange.to),
    });
    rangeStartRef.current = undefined;
    setMonth(parseCalendarDate(nextRange.from));
    onChange(nextRange);
  }

  function applyPreset(nextPeriod: Exclude<AnalyticsPeriod, "custom">) {
    applyRange(analyticsPeriodRange(nextPeriod, todayValue), nextPeriod);
  }

  function selectDay(day: Date, disabled: boolean | undefined) {
    if (disabled) return;

    const rangeStart = rangeStartRef.current;
    if (!rangeStart) {
      setPeriod("custom");
      setDraft({ from: day, to: undefined });
      rangeStartRef.current = day;
      return;
    }

    const from = day < rangeStart ? day : rangeStart;
    const to = day < rangeStart ? rangeStart : day;
    if (differenceInCalendarDays(to, from) > 365) {
      setDraft({ from: day, to: undefined });
      rangeStartRef.current = day;
      return;
    }

    applyRange({ from: format(from, "yyyy-MM-dd"), to: format(to, "yyyy-MM-dd") }, "custom");
  }

  function movePeriod(amount: -1 | 1) {
    applyRange(shiftAnalyticsPeriod(value, period, amount), period);
  }

  return (
    <div className="flex w-full items-center gap-1 sm:w-auto">
      <Button
        variant="outline"
        size="icon"
        aria-label={m.admin_analytics_previous_period()}
        title={m.admin_analytics_previous_period()}
        onClick={() => movePeriod(-1)}
      >
        <HugeiconsIcon icon={ArrowLeftIcon} />
      </Button>
      <Popover
        open={open}
        onOpenChange={(nextOpen) => {
          setOpen(nextOpen);
          if (nextOpen) {
            setDraft({ from: parseCalendarDate(value.from), to: parseCalendarDate(value.to) });
            rangeStartRef.current = undefined;
            setMonth(parseCalendarDate(value.from));
          }
        }}
      >
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              className="min-w-0 flex-1 justify-start sm:min-w-64 sm:flex-none"
            />
          }
        >
          <HugeiconsIcon icon={Calendar03Icon} data-icon="inline-start" />
          <span className="truncate">{formatDateRange(value)}</span>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-auto gap-0 p-0">
          <div className="p-3 pb-2">
            <ToggleGroup
              value={period === "custom" ? [] : [period]}
              onValueChange={(nextValue) => {
                const nextPeriod = nextValue[0];
                if (nextPeriod === "week" || nextPeriod === "month" || nextPeriod === "year") {
                  applyPreset(nextPeriod);
                }
              }}
              variant="outline"
              size="sm"
              spacing={0}
              className="grid w-full grid-cols-3"
              aria-label={m.admin_analytics_date_presets()}
            >
              <ToggleGroupItem value="week">{m.admin_analytics_this_week()}</ToggleGroupItem>
              <ToggleGroupItem value="month">{m.admin_analytics_this_month()}</ToggleGroupItem>
              <ToggleGroupItem value="year">{m.admin_analytics_this_year()}</ToggleGroupItem>
            </ToggleGroup>
          </div>
          <Separator />
          <Calendar
            mode="range"
            today={today}
            selected={draft}
            onSelect={(_nextRange, day, modifiers) => selectDay(day, modifiers.disabled)}
            locale={DATE_FNS_LOCALES[getLocale()]}
            month={month}
            onMonthChange={setMonth}
            className="mx-auto"
            autoFocus
          />
        </PopoverContent>
      </Popover>
      <Button
        variant="outline"
        size="icon"
        aria-label={m.admin_analytics_next_period()}
        title={m.admin_analytics_next_period()}
        onClick={() => movePeriod(1)}
      >
        <HugeiconsIcon icon={ArrowRightIcon} />
      </Button>
    </div>
  );
}

function topStackRadius(
  point: AdminAnalytics["points"][number],
  dataKey: keyof AdminAnalytics["points"][number],
  stackKeys: readonly (keyof AdminAnalytics["points"][number])[],
): number | [number, number, number, number] {
  const dataKeyIndex = stackKeys.indexOf(dataKey);
  const isVisible = Number(point[dataKey]) > 0;
  const hasVisibleSegmentAbove = stackKeys
    .slice(dataKeyIndex + 1)
    .some((key) => Number(point[key]) > 0);
  return isVisible && !hasVisibleSegmentAbove ? [4, 4, 0, 0] : 0;
}

function roundedStackShape({
  dataKey,
  stackKeys,
}: {
  dataKey: keyof AdminAnalytics["points"][number];
  stackKeys: readonly (keyof AdminAnalytics["points"][number])[];
}) {
  return (props: BarShapeProps) => (
    <Rectangle
      {...props}
      radius={topStackRadius(props.payload as AdminAnalytics["points"][number], dataKey, stackKeys)}
    />
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

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle>
          <H2 className="tabular-nums">{value}</H2>
        </CardTitle>
      </CardHeader>
    </Card>
  );
}

function SummaryGrid({ current }: { current: AdminAnalyticsTotals }) {
  const numberFormatter = NUMBER_FORMATTERS[getLocale()];
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <SummaryCard label={m.games_played()} value={numberFormatter.format(current.games)} />
      <SummaryCard
        label={m.admin_analytics_active_players()}
        value={numberFormatter.format(current.activePlayers)}
      />
      <SummaryCard
        label={m.total_playtime()}
        value={formatPlaytime(current.activePlaytimeSeconds)}
      />
      <SummaryCard
        label={m.admin_analytics_healthy_finish_rate()}
        value={PERCENT_FORMATTERS[getLocale()].format(current.healthyFinishRate)}
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
  const playerStackKeys = ["newPlayers", "returningPlayers"] as const;
  const outcomeConfig = {
    completed: { label: m.admin_analytics_completed(), color: "var(--chart-1)" },
    forfeit: { label: m.forfeit(), color: "var(--chart-2)" },
    mutualEnd: { label: m.admin_analytics_mutual_end(), color: "var(--chart-3)" },
    technicalAbort: { label: m.admin_analytics_technical_abort(), color: "var(--chart-4)" },
    abandoned: { label: m.admin_analytics_abandoned(), color: "var(--chart-5)" },
    inProgress: { label: m.admin_analytics_in_progress(), color: "var(--muted-foreground)" },
  } satisfies ChartConfig;
  const outcomeStackKeys = [
    "completed",
    "forfeit",
    "mutualEnd",
    "technicalAbort",
    "abandoned",
    "inProgress",
  ] as const;
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
            <Bar
              dataKey="newPlayers"
              stackId="players"
              fill="var(--color-newPlayers)"
              shape={roundedStackShape({ dataKey: "newPlayers", stackKeys: playerStackKeys })}
            />
            <Bar
              dataKey="returningPlayers"
              stackId="players"
              fill="var(--color-returningPlayers)"
              shape={roundedStackShape({ dataKey: "returningPlayers", stackKeys: playerStackKeys })}
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
            {outcomeStackKeys.map((dataKey) => (
              <Bar
                key={dataKey}
                dataKey={dataKey}
                stackId="outcomes"
                fill={`var(--color-${dataKey})`}
                shape={roundedStackShape({ dataKey, stackKeys: outcomeStackKeys })}
              />
            ))}
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
            <SummaryGrid current={data.current} />
            <AnalyticsCharts analytics={data} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

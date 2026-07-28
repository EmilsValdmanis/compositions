import {
  Alert02Icon,
  Bug01Icon,
  CheckmarkCircle02Icon,
  CodeIcon,
  Copy01Icon,
  EyeIcon,
  GameController03Icon,
  Layers01Icon,
  RefreshIcon,
  UserIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { GameCard } from "#/components/game/game-card";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "#/components/ui/empty";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "#/components/ui/pagination";
import { ScrollArea, ScrollBar } from "#/components/ui/scroll-area";
import { Separator } from "#/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "#/components/ui/sheet";
import { Skeleton } from "#/components/ui/skeleton";
import { Spinner } from "#/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table";
import { Caption, H3, P } from "#/components/typography";
import {
  adminBugReportDetailOptions,
  adminBugReportPageOptions,
  completeAdminBugReport,
  persistedGameStateSchema,
  type AdminBugReportDetail,
  type AdminBugReportPage,
  type PersistedGameState,
} from "#/lib/admin-bug-reports";
import { copyTextToClipboard } from "#/lib/clipboard";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { getLocale, type Locale } from "#/paraglide/runtime.js";

const CREATED_AT_FORMATTERS: Record<Locale, Intl.DateTimeFormat> = {
  en: new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Europe/Riga",
  }),
  lv: new Intl.DateTimeFormat("lv", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Europe/Riga",
  }),
};

const PHASE_LABELS = [
  m.admin_game_phase_lobby,
  m.admin_game_phase_in_progress,
  m.admin_game_phase_round_over,
  m.admin_game_phase_game_over,
] as const;

function formatCreatedAt(value: string) {
  return CREATED_AT_FORMATTERS[getLocale()].format(new Date(value));
}

function phaseLabel(phase: number) {
  return PHASE_LABELS[phase]?.() ?? m.admin_game_phase_unknown();
}

function withOccurrenceKeys<T>(items: T[], identify: (item: T) => string) {
  const counts = new Map<string, number>();

  return items.map((item) => {
    const identity = identify(item);
    const occurrence = (counts.get(identity) ?? 0) + 1;
    counts.set(identity, occurrence);
    return { item, key: JSON.stringify([identity, occurrence]) };
  });
}

function cardIdentity(card: PersistedGameState["drawPile"][number]) {
  return JSON.stringify([card.rank ?? null, card.suit ?? null, card.isJoker ?? false]);
}

function compositionIdentity(composition: PersistedGameState["activeCompositions"][number]) {
  return JSON.stringify([composition.variant, composition.cards.map((card) => cardIdentity(card))]);
}

function DetailItem({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <Caption>{label}</Caption>
      <P size="sm" className="min-w-0 break-words font-medium">
        {value}
      </P>
    </div>
  );
}

function ReportMetadata({ report }: { report: AdminBugReportDetail }) {
  return (
    <div className="grid gap-4 rounded-2xl bg-muted/50 p-4 sm:grid-cols-2">
      <DetailItem label={m.admin_report_id()} value={report.id} />
      <DetailItem label={m.admin_room()} value={report.roomCode} />
      <DetailItem label={m.admin_reporter_player()} value={report.reporterPlayerId} />
      <DetailItem
        label={m.admin_reporter_user()}
        value={report.reporterUserId ?? m.admin_anonymous_player()}
      />
      <DetailItem label={m.round()} value={report.round} />
      <DetailItem label={m.turn()} value={report.turn} />
      <DetailItem label={m.admin_reported_at()} value={formatCreatedAt(report.createdAt)} />
      <DetailItem
        label={m.admin_abort_requested()}
        value={report.requestedAbort ? m.yes() : m.no()}
      />
    </div>
  );
}

function StateMetric({
  label,
  value,
  icon,
}: {
  label: string;
  value: React.ReactNode;
  icon: Parameters<typeof HugeiconsIcon>[0]["icon"];
}) {
  return (
    <div className="flex min-w-0 items-center gap-3 rounded-2xl bg-muted/50 p-3">
      <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-background text-primary shadow-sm">
        <HugeiconsIcon icon={icon} aria-hidden="true" />
      </span>
      <div className="min-w-0">
        <Caption>{label}</Caption>
        <P size="sm" className="truncate font-semibold tabular-nums">
          {value}
        </P>
      </div>
    </div>
  );
}

function CardCollection({
  title,
  cards,
  description,
}: {
  title: string;
  cards: PersistedGameState["drawPile"];
  description?: string;
}) {
  return (
    <Card size="sm" className="w-full min-w-0 max-w-full">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
        <CardAction>
          <Badge variant="outline">{cards.length}</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="min-w-0">
        {cards.length > 0 ? (
          <div className="flex w-full min-w-0 max-w-full gap-2 overflow-x-auto overscroll-x-contain pb-1">
            {withOccurrenceKeys(cards, cardIdentity).map(({ item: card, key }) => (
              <GameCard key={key} card={card} size="compact" />
            ))}
          </div>
        ) : (
          <P size="sm" className="text-muted-foreground">
            {m.admin_no_cards()}
          </P>
        )}
      </CardContent>
    </Card>
  );
}

function VisualGameState({ state }: { state: PersistedGameState }) {
  const activePlayer = state.players[state.turn.playerIndex];
  const topDiscard = state.discardPile[0];

  return (
    <div className="flex w-full min-w-0 max-w-full flex-col gap-4">
      <div className="grid min-w-0 gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <StateMetric
          label={m.admin_game_phase()}
          value={phaseLabel(state.phase)}
          icon={GameController03Icon}
        />
        <StateMetric
          label={m.admin_round_turn()}
          value={`${state.round} · ${state.turn.number}`}
          icon={RefreshIcon}
        />
        <StateMetric
          label={m.admin_active_player()}
          value={activePlayer?.id ?? m.admin_unknown_player()}
          icon={UserIcon}
        />
        <StateMetric
          label={m.admin_draw_pile()}
          value={state.drawPile.length}
          icon={Layers01Icon}
        />
      </div>

      <Card className="min-w-0">
        <CardHeader>
          <CardTitle>{m.admin_table_compositions()}</CardTitle>
          <CardDescription>
            {m.admin_table_compositions_description({
              count: state.activeCompositions.length,
            })}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex min-w-0 flex-col gap-3">
          {state.activeCompositions.length > 0 ? (
            withOccurrenceKeys(state.activeCompositions, compositionIdentity).map(
              ({ item: composition, key }, index) => (
                <CardCollection
                  key={key}
                  title={`${m.admin_composition()} ${index + 1}`}
                  description={composition.variant}
                  cards={composition.cards}
                />
              ),
            )
          ) : (
            <P size="sm" className="text-muted-foreground">
              {m.admin_no_compositions()}
            </P>
          )}
        </CardContent>
      </Card>

      <div className="grid min-w-0 gap-3 xl:grid-cols-2">
        {state.players.map((player, index) => (
          <Card key={player.id} size="sm" className="min-w-0">
            <CardHeader>
              <CardTitle className="flex flex-wrap items-center gap-2">
                <span>{player.id}</span>
                {index === state.turn.playerIndex ? <Badge>{m.admin_active()}</Badge> : null}
                {player.forfeited ? <Badge variant="destructive">{m.forfeit()}</Badge> : null}
              </CardTitle>
              <CardDescription>
                {m.admin_player_points({
                  total: player.totalPoints,
                  gained: player.pointsGained,
                })}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex w-full min-w-0 max-w-full gap-2 overflow-x-auto overscroll-x-contain pb-1">
                {withOccurrenceKeys(player.hand, cardIdentity).map(({ item: card, key }) => (
                  <GameCard key={key} card={card} size="compact" />
                ))}
              </div>
            </CardContent>
            <CardFooter className="flex-wrap gap-2">
              <Badge variant="outline">
                {player.hasOpened ? m.admin_opened() : m.admin_not_opened()}
              </Badge>
              <Badge variant="secondary">{m.admin_card_count({ count: player.hand.length })}</Badge>
            </CardFooter>
          </Card>
        ))}
      </div>

      <div className="grid min-w-0 gap-3 sm:grid-cols-2">
        <CardCollection
          title={m.admin_discard_pile()}
          cards={state.discardPile}
          description={topDiscard ? m.admin_top_card_visible() : m.admin_no_discard_card()}
        />
        <Card size="sm" className="min-w-0">
          <CardHeader>
            <CardTitle>{m.admin_state_flags()}</CardTitle>
          </CardHeader>
          <CardContent className="flex min-w-0 flex-col gap-3">
            <div className="flex min-w-0 flex-wrap gap-2">
              <Badge variant={state.turn.hasDrawn ? "default" : "outline"}>
                {state.turn.hasDrawn ? m.admin_has_drawn() : m.admin_has_not_drawn()}
              </Badge>
              <Badge variant={state.turn.mustUseDiscardDraw ? "destructive" : "secondary"}>
                {state.turn.mustUseDiscardDraw ? m.admin_must_use_discard() : m.admin_free_play()}
              </Badge>
              <Badge variant="outline">
                {state.gameMode === "quick" ? m.quick_game() : m.ranked_full()}
              </Badge>
            </div>
            <Separator />
            <DetailItem label={m.admin_snapshot_version()} value={state.version} />
            <DetailItem label={m.admin_dealer_index()} value={state.dealerIndex} />
            <DetailItem label={m.admin_round_winner_index()} value={state.roundWinnerIndex} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function GameStateViewer({ report }: { report: AdminBugReportDetail }) {
  const parsedState = persistedGameStateSchema.safeParse(report.gameState);
  const json = JSON.stringify(report.gameState, null, 2);

  async function copyJson() {
    try {
      await copyTextToClipboard(json);
      toast.success(m.admin_json_copied());
    } catch {
      toast.error(m.admin_json_copy_error());
    }
  }

  return (
    <Tabs defaultValue="visual" className="min-h-0 min-w-0 max-w-full">
      <TabsList>
        <TabsTrigger value="visual">
          <HugeiconsIcon icon={EyeIcon} data-icon="inline-start" />
          {m.admin_visual_state()}
        </TabsTrigger>
        <TabsTrigger value="json">
          <HugeiconsIcon icon={CodeIcon} data-icon="inline-start" />
          {m.admin_raw_json()}
        </TabsTrigger>
      </TabsList>
      <TabsContent value="visual" className="min-w-0 pt-2">
        {parsedState.success ? (
          <VisualGameState state={parsedState.data} />
        ) : (
          <Alert variant="destructive">
            <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
            <AlertTitle>{m.admin_state_unreadable()}</AlertTitle>
            <AlertDescription>{m.admin_state_unreadable_description()}</AlertDescription>
          </Alert>
        )}
      </TabsContent>
      <TabsContent value="json" className="min-w-0 pt-2">
        <Card size="sm" className="min-w-0">
          <CardHeader className="border-b">
            <CardTitle>{m.admin_raw_json()}</CardTitle>
            <CardDescription>application/json</CardDescription>
            <CardAction>
              <Button type="button" variant="outline" size="sm" onClick={() => void copyJson()}>
                <HugeiconsIcon icon={Copy01Icon} data-icon="inline-start" />
                {m.admin_copy_json()}
              </Button>
            </CardAction>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-96 min-w-0 max-w-full rounded-2xl bg-muted/50">
              <pre className="min-w-max p-4 font-mono text-xs/5">{json}</pre>
              <ScrollBar orientation="horizontal" />
            </ScrollArea>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}

function ReportDetailSkeleton() {
  return (
    <div className="flex flex-col gap-4 p-4 sm:p-6">
      <Skeleton className="h-24 w-full rounded-2xl" />
      <Skeleton className="h-40 w-full rounded-2xl" />
      <Skeleton className="h-72 w-full rounded-2xl" />
    </div>
  );
}

function ReportDetailSheet({
  reportId,
  onOpenChange,
  onCompleted,
}: {
  reportId: string | null;
  onOpenChange: (open: boolean) => void;
  onCompleted: () => void;
}) {
  const queryClient = useQueryClient();
  const { data, isError, isPending, refetch } = useQuery(adminBugReportDetailOptions(reportId));
  const completeReport = useMutation({
    mutationFn: (id: string) => completeAdminBugReport({ data: id }),
    onSuccess: async (_, completedReportId) => {
      queryClient.removeQueries({ queryKey: ["admin", "bug-report", completedReportId] });
      await queryClient.invalidateQueries({ queryKey: ["admin", "bug-reports"] });
      toast.success(m.admin_report_completed());
      onCompleted();
    },
    onError: () => {
      toast.error(m.admin_report_complete_error());
    },
  });

  return (
    <Sheet open={reportId !== null} onOpenChange={onOpenChange}>
      <SheetContent className="w-full! max-w-[58rem]! min-w-0 overflow-hidden">
        <SheetHeader className="shrink-0 border-b p-4 pr-14 sm:p-6 sm:pr-16">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex min-w-0 flex-col gap-1.5">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={data?.requestedAbort ? "destructive" : "secondary"}>
                  {data?.requestedAbort ? m.admin_abort_requested() : m.admin_report_only()}
                </Badge>
                {data ? <Badge variant="outline">{data.roomCode}</Badge> : null}
              </div>
              <SheetTitle>{m.admin_bug_report_detail()}</SheetTitle>
              <SheetDescription>
                {data ? formatCreatedAt(data.createdAt) : m.admin_loading_report()}
              </SheetDescription>
            </div>
            <Button
              size="sm"
              disabled={!data || completeReport.isPending}
              onClick={() => {
                if (data) completeReport.mutate(data.id);
              }}
            >
              {completeReport.isPending ? (
                <Spinner data-icon="inline-start" />
              ) : (
                <HugeiconsIcon icon={CheckmarkCircle02Icon} data-icon="inline-start" />
              )}
              {m.complete()}
            </Button>
          </div>
        </SheetHeader>

        <ScrollArea className="min-h-0 min-w-0 flex-1 overscroll-contain">
          {isPending ? (
            <ReportDetailSkeleton />
          ) : isError || !data ? (
            <div className="p-4 sm:p-6">
              <Alert variant="destructive">
                <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
                <AlertTitle>{m.admin_report_load_error()}</AlertTitle>
                <AlertDescription className="flex flex-col items-start gap-3">
                  <span>{m.admin_report_load_error_description()}</span>
                  <Button variant="outline" size="sm" onClick={() => void refetch()}>
                    <HugeiconsIcon icon={RefreshIcon} data-icon="inline-start" />
                    {m.try_again()}
                  </Button>
                </AlertDescription>
              </Alert>
            </div>
          ) : (
            <div className="flex w-full min-w-0 max-w-full flex-col gap-6 overflow-x-hidden p-4 sm:p-6">
              <section className="flex min-w-0 flex-col gap-3">
                <Caption className="uppercase tracking-[0.16em]">{m.admin_report()}</Caption>
                <P size="lg" className="text-pretty">
                  {data.description}
                </P>
              </section>
              <ReportMetadata report={data} />
              <Separator />
              <section className="flex min-w-0 flex-col gap-3">
                <div>
                  <H3>{m.admin_game_state()}</H3>
                  <P size="sm" className="text-muted-foreground">
                    {m.admin_game_state_description()}
                  </P>
                </div>
                <GameStateViewer report={data} />
              </section>
            </div>
          )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}

function ReportsTable({
  reports,
  isFetching,
  onSelect,
}: {
  reports: AdminBugReportPage;
  isFetching: boolean;
  onSelect: (reportId: string) => void;
}) {
  return (
    <Table
      aria-label={m.admin_bug_reports()}
      className={cn("min-w-4xl", isFetching && "opacity-60")}
      containerClassName="max-w-full overscroll-x-contain"
    >
      <TableHeader>
        <TableRow className="hover:bg-transparent">
          <TableHead>{m.admin_reported_at()}</TableHead>
          <TableHead>{m.admin_room()}</TableHead>
          <TableHead>{m.admin_description()}</TableHead>
          <TableHead>{m.admin_position()}</TableHead>
          <TableHead>{m.admin_kind()}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {reports.reports.map((report) => (
          <TableRow key={report.id} className="cursor-pointer" onClick={() => onSelect(report.id)}>
            <TableCell className="font-medium">{formatCreatedAt(report.createdAt)}</TableCell>
            <TableCell>
              <Badge variant="outline">{report.roomCode}</Badge>
            </TableCell>
            <TableCell className="max-w-md">
              <span className="block truncate">{report.description}</span>
            </TableCell>
            <TableCell className="tabular-nums">
              {m.admin_round_turn_short({ round: report.round, turn: report.turn })}
            </TableCell>
            <TableCell>
              <Badge variant={report.requestedAbort ? "destructive" : "secondary"}>
                {report.requestedAbort ? m.admin_abort() : m.admin_report_only()}
              </Badge>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function AdminBugReportsPage({ initialPage }: { initialPage: AdminBugReportPage }) {
  const [page, setPage] = useState(1);
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);
  const { data, isError, isFetching, isPending, refetch } = useQuery({
    ...adminBugReportPageOptions(page),
    initialData: page === 1 ? initialPage : undefined,
    placeholderData: keepPreviousData,
  });
  const reports = data ?? initialPage;

  return (
    <section className="mx-auto w-full min-w-0 max-w-7xl">
      <Card className="min-h-96 min-w-0">
        <CardHeader>
          <CardTitle>{m.admin_incident_archive()}</CardTitle>
          <CardDescription>{m.admin_incident_archive_description()}</CardDescription>
        </CardHeader>

        {isError ? (
          <CardContent>
            <Alert variant="destructive">
              <HugeiconsIcon icon={Alert02Icon} aria-hidden="true" />
              <AlertTitle>{m.admin_reports_load_error()}</AlertTitle>
              <AlertDescription className="flex flex-col items-start gap-3">
                <span>{m.admin_reports_load_error_description()}</span>
                <Button variant="outline" size="sm" onClick={() => void refetch()}>
                  <HugeiconsIcon icon={RefreshIcon} data-icon="inline-start" />
                  {m.try_again()}
                </Button>
              </AlertDescription>
            </Alert>
          </CardContent>
        ) : isPending ? (
          <CardContent className="grid min-h-64 place-items-center">
            <Spinner />
          </CardContent>
        ) : reports.reports.length === 0 ? (
          <Empty className="min-h-64 border-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HugeiconsIcon icon={Bug01Icon} aria-hidden="true" />
              </EmptyMedia>
              <EmptyTitle>{m.admin_no_bug_reports()}</EmptyTitle>
              <EmptyDescription>{m.admin_no_bug_reports_description()}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ReportsTable reports={reports} isFetching={isFetching} onSelect={setSelectedReportId} />
        )}

        {!isPending && reports.totalPages > 1 ? (
          <CardFooter className="flex-col justify-between gap-3 border-t sm:flex-row">
            <Caption className="tabular-nums">
              {m.page_of({ page: reports.page, total: reports.totalPages })}
            </Caption>
            <Pagination className="mx-0 w-auto">
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    href="#"
                    text={m.previous()}
                    aria-label={m.previous()}
                    aria-disabled={page <= 1 || isFetching}
                    tabIndex={page <= 1 || isFetching ? -1 : undefined}
                    onClick={(event) => {
                      event.preventDefault();
                      if (page > 1 && !isFetching) setPage((current) => current - 1);
                    }}
                  />
                </PaginationItem>
                <PaginationItem>
                  <PaginationNext
                    href="#"
                    text={m.next()}
                    aria-label={m.next()}
                    aria-disabled={page >= reports.totalPages || isFetching}
                    tabIndex={page >= reports.totalPages || isFetching ? -1 : undefined}
                    onClick={(event) => {
                      event.preventDefault();
                      if (page < reports.totalPages && !isFetching) {
                        setPage((current) => current + 1);
                      }
                    }}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </CardFooter>
        ) : null}
      </Card>

      <ReportDetailSheet
        reportId={selectedReportId}
        onOpenChange={(open) => {
          if (!open) setSelectedReportId(null);
        }}
        onCompleted={() => {
          setSelectedReportId(null);
          if (reports.reports.length === 1 && page > 1) {
            setPage((current) => current - 1);
          }
        }}
      />
    </section>
  );
}

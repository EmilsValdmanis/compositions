import { createFileRoute, redirect } from "@tanstack/react-router";
import { LeaderboardPage } from "#/components/routes/leaderboard-page";
import {
  DEFAULT_LEADERBOARD_METRIC,
  DEFAULT_LEADERBOARD_SCOPE,
  leaderboardInfiniteOptions,
} from "#/lib/leaderboard";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";

export const Route = createFileRoute("/_protected/leaderboard")({
  loader: async ({ context }) => {
    const playerId = context.session?.user.id;
    if (!playerId) throw redirect({ to: "/sign-in" });
    await context.queryClient.prefetchInfiniteQuery(
      leaderboardInfiniteOptions(playerId, DEFAULT_LEADERBOARD_METRIC, DEFAULT_LEADERBOARD_SCOPE),
    );
  },
  head: () => ({
    meta: [{ title: pageTitle(m.leaderboard()) }],
  }),
  component: LeaderboardRoute,
});

function LeaderboardRoute() {
  const { session } = Route.useRouteContext();
  return <LeaderboardPage playerId={session!.user.id} />;
}

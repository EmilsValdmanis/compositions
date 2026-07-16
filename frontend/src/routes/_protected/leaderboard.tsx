import { createFileRoute, redirect } from "@tanstack/react-router";
import { LeaderboardPage } from "#/components/routes/leaderboard-page";
import { leaderboardInfiniteOptions } from "#/lib/leaderboard";
import { m } from "#/paraglide/messages.js";

export const Route = createFileRoute("/_protected/leaderboard")({
  loader: async ({ context }) => {
    const playerId = context.session?.user.id;
    if (!playerId) throw redirect({ to: "/sign-in" });
    await context.queryClient.prefetchInfiniteQuery(leaderboardInfiniteOptions(playerId));
  },
  head: () => ({
    meta: [{ title: `${m.leaderboard()} · ${m.app_name()}` }],
  }),
  component: LeaderboardRoute,
});

function LeaderboardRoute() {
  const { session } = Route.useRouteContext();
  return <LeaderboardPage playerId={session!.user.id} />;
}

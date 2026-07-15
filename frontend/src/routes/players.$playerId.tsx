import { createFileRoute, notFound } from "@tanstack/react-router";
import { PlayerProfilePage } from "#/components/routes/player-profile-page";
import { AppNavigation } from "#/components/routes/protected-layout";
import { getPlayerGameHistory, getPlayerProfile } from "#/lib/player-profile";
import { createSocialMeta } from "#/lib/social-meta";
import { m } from "#/paraglide/messages.js";
import { localizeHref } from "#/paraglide/runtime.js";

function profileDescription(profile: Awaited<ReturnType<typeof getPlayerProfile>> | undefined) {
  if (!profile || profile.gamesPlayed === 0) {
    return profile
      ? m.profile_first_game_description({ name: profile.name })
      : m.profile_generic_description();
  }

  const winRate = Math.round((profile.gamesWon / profile.gamesPlayed) * 100);
  return m.profile_stats_description({
    winsText: m.profile_wins_count({ count: profile.gamesWon }),
    gamesText: m.profile_games_count({ count: profile.gamesPlayed }),
    winRate,
    compositionsText: m.profile_compositions_count({ count: profile.compositionsCreated }),
  });
}

export const Route = createFileRoute("/players/$playerId")({
  loader: async ({ params }) => {
    const [profileResult, historyResult] = await Promise.allSettled([
      getPlayerProfile({ data: params.playerId }),
      getPlayerGameHistory({
        data: { playerId: params.playerId, page: 1, pageSize: 10 },
      }),
    ]);
    if (profileResult.status === "rejected") throw profileResult.reason;
    const profile = profileResult.value;
    if (!profile) throw notFound();
    if (historyResult.status === "rejected") throw historyResult.reason;
    return { profile, history: historyResult.value };
  },
  head: ({ loaderData, match }) => {
    const profile = loaderData?.profile;
    const title = profile ? `${profile.name} · ${m.app_name()}` : m.profile_title();
    const description = profileDescription(profile);
    const origin = match.context.siteOrigin;
    const url = profile
      ? `${origin}${localizeHref(`/players/${profile.id}`)}`
      : `${origin}${localizeHref("/")}`;

    return {
      meta: [{ title }, ...createSocialMeta({ title, description, origin, type: "profile", url })],
      links: [{ rel: "canonical", href: url }],
    };
  },
  component: PlayerProfileRoute,
});

function PlayerProfileRoute() {
  const { profile, history } = Route.useLoaderData();
  const { session } = Route.useRouteContext();

  return (
    <>
      <AppNavigation />
      <main className="flex min-h-0 w-full flex-1 flex-col p-4 md:p-6">
        <PlayerProfilePage
          profile={profile}
          initialHistory={history}
          isOwnProfile={session?.user.id === profile.id}
        />
      </main>
    </>
  );
}

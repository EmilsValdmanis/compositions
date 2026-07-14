import { createFileRoute, notFound } from "@tanstack/react-router";
import { PlayerProfilePage } from "#/components/routes/player-profile-page";
import { AppNavigation } from "#/components/routes/protected-layout";
import { getPlayerProfile } from "#/lib/player-profile";
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
    const profile = await getPlayerProfile({ data: params.playerId });
    if (!profile) throw notFound();
    return profile;
  },
  head: ({ loaderData, match }) => {
    const title = loaderData ? `${loaderData.name} · ${m.app_name()}` : m.profile_title();
    const description = profileDescription(loaderData);
    const origin = match.context.siteOrigin;
    const url = loaderData
      ? `${origin}${localizeHref(`/players/${loaderData.id}`)}`
      : `${origin}${localizeHref("/")}`;

    return {
      meta: [{ title }, ...createSocialMeta({ title, description, origin, type: "profile", url })],
      links: [{ rel: "canonical", href: url }],
    };
  },
  component: PlayerProfileRoute,
});

function PlayerProfileRoute() {
  const profile = Route.useLoaderData();
  const { session } = Route.useRouteContext();

  return (
    <>
      <AppNavigation />
      <main className="flex min-h-0 w-full flex-1 flex-col p-4 md:p-6">
        <PlayerProfilePage profile={profile} isOwnProfile={session?.user.id === profile.id} />
      </main>
    </>
  );
}

import { createFileRoute, notFound } from "@tanstack/react-router";
import { PlayerProfilePage } from "#/components/routes/player-profile-page";
import { AppNavigation } from "#/components/routes/protected-layout";
import { getPlayerProfile } from "#/lib/player-profile";
import { createSocialMeta } from "#/lib/social-meta";

function profileDescription(profile: Awaited<ReturnType<typeof getPlayerProfile>> | undefined) {
  if (!profile || profile.gamesPlayed === 0) {
    return profile
      ? `${profile.name} is getting ready for their first ranked game of Compositions.`
      : "View player statistics and ranked history in Compositions.";
  }

  const winRate = Math.round((profile.gamesWon / profile.gamesPlayed) * 100);
  return `${profile.gamesWon} wins from ${profile.gamesPlayed} games (${winRate}% win rate) and ${profile.compositionsCreated} compositions created.`;
}

export const Route = createFileRoute("/players/$playerId")({
  loader: async ({ params }) => {
    const profile = await getPlayerProfile({ data: params.playerId });
    if (!profile) throw notFound();
    return profile;
  },
  head: ({ loaderData, match }) => {
    const title = loaderData ? `${loaderData.name} · Compositions` : "Player · Compositions";
    const description = profileDescription(loaderData);
    const origin = match.context.siteOrigin;
    const url = loaderData ? `${origin}/players/${loaderData.id}` : origin;

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

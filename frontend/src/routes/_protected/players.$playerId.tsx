import { createFileRoute, notFound } from "@tanstack/react-router";
import { PlayerProfilePage } from "#/components/routes/player-profile-page";
import { getPlayerProfile } from "#/lib/player-profile";

export const Route = createFileRoute("/_protected/players/$playerId")({
  loader: async ({ params }) => {
    const profile = await getPlayerProfile({ data: params.playerId });
    if (!profile) throw notFound();
    return profile;
  },
  head: ({ loaderData }) => ({
    meta: [{ title: loaderData ? `${loaderData.name} · Compositions` : "Player · Compositions" }],
  }),
  component: PlayerProfileRoute,
});

function PlayerProfileRoute() {
  return <PlayerProfilePage profile={Route.useLoaderData()} />;
}

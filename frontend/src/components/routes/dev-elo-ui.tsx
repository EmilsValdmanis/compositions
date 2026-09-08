import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { LeaderboardPage } from "./leaderboard-page";
import { leaderboardMetricSchema, type LeaderboardPlayer } from "#/lib/leaderboard";

const viewerId = "00000000-0000-4000-8000-000000000004";

// In-memory fixtures for /dev-ui?view=elo; no player data is written to the API.
function previewQueryClient(state?: "empty" | "placement") {
  const client = new QueryClient({
    defaultOptions: {
      queries: { refetchOnMount: false, refetchOnWindowFocus: false, refetchOnReconnect: false },
    },
  });
  const players: LeaderboardPlayer[] = [
    ["Alex Morgan", 2348, "grandmaster"],
    ["Sofia Andersson", 2086, "master"],
    ["Emīls Kalniņš", 1874, "diamond"],
    ["Casey Taylor", 1682, "platinum"],
    ["Maya Chen", 1456, "gold"],
    ["Liam Wilson", 1264, "silver"],
    ["Anna Bērziņa", 1068, "bronze"],
  ].map(([name, rating, tier], i) => ({
    playerId: `00000000-0000-4000-8000-${String(i + 1).padStart(12, "0")}`,
    rank: i + 1,
    score: Number(rating),
    name: String(name),
    tier: tier as LeaderboardPlayer["tier"],
    imageUrl: "",
    wins: 70 - i * 8,
    gamesPlayed: 100 - i * 7,
    roundsWon: 250 - i * 20,
    pointsInflicted: 6000 - i * 500,
    totalPlaytimeSeconds: 86400 - i * 3600,
  }));
  for (const scope of ["friends", "global"] as const) {
    for (const metric of leaderboardMetricSchema.options) {
      const rows = players.map((player) => ({
        ...player,
        score: {
          elo: player.score,
          wins: player.wins,
          games: player.gamesPlayed,
          rounds: player.roundsWon,
          points: player.pointsInflicted,
          playtime: player.totalPlaytimeSeconds,
        }[metric],
      }));
      client.setQueryData(["leaderboard", viewerId, metric, scope], {
        pages: [
          {
            metric,
            scope,
            players: state === "empty" ? [] : state === "placement" ? rows.slice(0, 3) : rows,
            placement: state === "empty" ? null : rows[3],
            nextCursor: null,
            reset: false,
          },
        ],
        pageParams: [null],
      });
    }
  }
  return client;
}

export function DevEloUi({ state }: { state?: "empty" | "placement" }) {
  const [client] = useState(() => previewQueryClient(state));
  return (
    <QueryClientProvider client={client}>
      <LeaderboardPage playerId={viewerId} />
    </QueryClientProvider>
  );
}

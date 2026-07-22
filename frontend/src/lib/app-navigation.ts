import { useRouterState } from "@tanstack/react-router";

export type AppPage = "lobby" | "leaderboard" | "dev-ui" | "profile" | "unknown";

function appPageFromRouteId(routeId: string | undefined): AppPage {
  switch (routeId) {
    case "/_protected/":
      return "lobby";
    case "/_protected/leaderboard":
      return "leaderboard";
    case "/_protected/dev-ui":
      return "dev-ui";
    case "/players/$playerId":
      return "profile";
    default:
      return "unknown";
  }
}

export function useAppPage() {
  const routeId = useRouterState({
    select: (state) => state.matches.at(-1)?.routeId,
  });

  return appPageFromRouteId(routeId);
}

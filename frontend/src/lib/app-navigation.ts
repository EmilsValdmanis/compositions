import { useRouterState } from "@tanstack/react-router";
import { m } from "#/paraglide/messages.js";

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

export function appPageLabel(page: AppPage) {
  switch (page) {
    case "lobby":
      return m.lobby();
    case "leaderboard":
      return m.leaderboard();
    case "dev-ui":
      return m.dev_ui();
    case "profile":
      return m.profile();
    default:
      return m.app_name();
  }
}

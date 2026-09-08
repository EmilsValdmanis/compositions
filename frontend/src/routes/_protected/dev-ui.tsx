import { createFileRoute, redirect } from "@tanstack/react-router";
import { DevEloUi } from "#/components/routes/dev-elo-ui";
import { DevGameUi } from "#/components/routes/dev-game-ui";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";

export const Route = createFileRoute("/_protected/dev-ui")({
  validateSearch: (
    search: Record<string, unknown>,
  ): { view?: "elo"; state?: "empty" | "placement" } => ({
    view: search.view === "elo" ? "elo" : undefined,
    state: search.state === "empty" || search.state === "placement" ? search.state : undefined,
  }),
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: "/" });
    }
  },
  head: () => ({
    meta: [{ title: pageTitle(m.dev_ui()) }],
  }),
  component: DevUiRoute,
});

function DevUiRoute() {
  const { view, state } = Route.useSearch();
  return view === "elo" ? <DevEloUi key={state} state={state} /> : <DevGameUi />;
}

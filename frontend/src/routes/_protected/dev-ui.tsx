import { createFileRoute, redirect } from "@tanstack/react-router";
import { DevGameUi } from "#/components/routes/dev-game-ui";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";

export const Route = createFileRoute("/_protected/dev-ui")({
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: "/" });
    }
  },
  head: () => ({
    meta: [{ title: pageTitle(m.dev_ui()) }],
  }),
  component: DevGameUi,
});

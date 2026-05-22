import { createFileRoute, redirect } from "@tanstack/react-router";
import { DevGameUi } from "#/components/routes/dev-game-ui";

export const Route = createFileRoute("/_protected/dev-ui")({
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: "/" });
    }
  },
  component: DevGameUi,
});

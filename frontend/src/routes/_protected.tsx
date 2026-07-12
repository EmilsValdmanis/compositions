import { createFileRoute, redirect } from "@tanstack/react-router";
import { ProtectedLayout } from "#/components/routes/protected-layout";

export const Route = createFileRoute("/_protected")({
  beforeLoad: ({ context, location }) => {
    const isDevUi = import.meta.env.DEV && location.pathname === "/dev-ui";

    if (!context.session && !isDevUi) {
      throw redirect({
        to: "/sign-in",
        search: { returnTo: location.href },
      });
    }
  },
  component: ProtectedLayout,
});

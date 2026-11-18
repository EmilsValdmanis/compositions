import { createFileRoute, redirect } from "@tanstack/react-router";
import { ProtectedLayout } from "#/components/routes/protected-layout";

export const Route = createFileRoute("/_protected")({
  beforeLoad: ({ context, location }) => {
    if (!context.session) {
      throw redirect({
        to: "/sign-in",
        search: { returnTo: location.href },
      });
    }
    return { session: context.session };
  },
  component: ProtectedLayout,
});

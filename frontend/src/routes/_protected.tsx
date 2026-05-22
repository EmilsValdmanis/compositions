import { createFileRoute, redirect } from "@tanstack/react-router";
import { ProtectedLayout } from "#/components/routes/protected-layout";

export const Route = createFileRoute("/_protected")({
  beforeLoad: ({ context }) => {
    if (!context.session) {
      throw redirect({ to: "/sign-in" });
    }
  },
  component: ProtectedLayout,
});

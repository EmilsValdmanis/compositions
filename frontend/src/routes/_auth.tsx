import { createFileRoute, redirect } from "@tanstack/react-router";
import { AuthLayout } from "#/components/routes/auth-layout";

export const Route = createFileRoute("/_auth")({
  beforeLoad: ({ context }) => {
    if (context.session) {
      throw redirect({ to: "/" });
    }
  },
  component: AuthLayout,
});

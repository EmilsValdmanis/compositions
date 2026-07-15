import { createFileRoute } from "@tanstack/react-router";
import { SignInPage } from "#/components/routes/sign-in-page";
import * as z from "zod";

export const Route = createFileRoute("/_auth/sign-in")({
  validateSearch: z.object({
    returnTo: z.string().startsWith("/").optional(),
  }),
  component: SignInRoute,
});

function SignInRoute() {
  const { returnTo } = Route.useSearch();
  return <SignInPage returnTo={returnTo} />;
}

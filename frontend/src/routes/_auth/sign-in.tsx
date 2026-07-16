import { createFileRoute } from "@tanstack/react-router";
import { SignInPage } from "#/components/routes/sign-in-page";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";
import * as z from "zod";

export const Route = createFileRoute("/_auth/sign-in")({
  validateSearch: z.object({
    returnTo: z.string().startsWith("/").optional(),
  }),
  head: () => ({
    meta: [{ title: pageTitle(m.sign_in()) }],
  }),
  component: SignInRoute,
});

function SignInRoute() {
  const { returnTo } = Route.useSearch();
  return <SignInPage returnTo={returnTo} />;
}

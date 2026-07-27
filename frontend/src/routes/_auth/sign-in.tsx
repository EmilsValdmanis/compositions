import { createFileRoute, redirect } from "@tanstack/react-router";
import { SignInPage } from "#/components/routes/sign-in-page";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";
import * as z from "zod";

const internalReturnTo = z
  .string()
  .refine(
    (value) => value.startsWith("/") && !value.startsWith("//") && !value.includes("\\"),
    "returnTo must be an internal path",
  );

export const Route = createFileRoute("/_auth/sign-in")({
  validateSearch: z.object({
    returnTo: internalReturnTo.optional(),
  }),
  beforeLoad: ({ context, search }) => {
    if (context.session) {
      throw redirect({ href: search.returnTo ?? "/" });
    }
  },
  head: () => ({
    meta: [{ title: pageTitle(m.sign_in()) }],
  }),
  component: SignInRoute,
});

function SignInRoute() {
  const { returnTo } = Route.useSearch();
  return <SignInPage returnTo={returnTo} />;
}

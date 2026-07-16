import { createFileRoute } from "@tanstack/react-router";
import { ProtectedHome } from "#/components/routes/protected-home";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";
import * as z from "zod";

export const Route = createFileRoute("/_protected/")({
  validateSearch: z.object({
    room: z
      .string()
      .trim()
      .min(1)
      .max(6)
      .transform((value) => value.toUpperCase())
      .optional(),
  }),
  head: () => ({
    meta: [{ title: pageTitle(m.start()) }],
  }),
  component: ProtectedHome,
});

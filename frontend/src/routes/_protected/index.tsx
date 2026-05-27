import { createFileRoute } from "@tanstack/react-router";
import { ProtectedHome } from "#/components/routes/protected-home";
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
  component: ProtectedHome,
});

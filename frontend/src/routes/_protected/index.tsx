import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { ProtectedHome } from "#/components/routes/protected-home";

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

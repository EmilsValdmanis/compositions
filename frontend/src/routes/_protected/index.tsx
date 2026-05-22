import { createFileRoute } from "@tanstack/react-router";
import { ProtectedHome } from "#/components/routes/protected-home";

export const Route = createFileRoute("/_protected/")({
  component: ProtectedHome,
});

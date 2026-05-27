import { createFileRoute } from "@tanstack/react-router";
import { ProtectedHome } from "#/components/routes/protected-home";

type ProtectedHomeSearch = {
  room?: string;
};

function validateProtectedHomeSearch(search: Record<string, unknown>): ProtectedHomeSearch {
  const room = typeof search.room === "string" ? search.room.trim().toUpperCase() : "";

  if (room.length === 0 || room.length > 6) {
    return {};
  }

  return { room };
}

export const Route = createFileRoute("/_protected/")({
  validateSearch: validateProtectedHomeSearch,
  component: ProtectedHome,
});

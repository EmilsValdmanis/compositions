import { ModeToggle } from "#/components/mode-toggle";
import { ServerStatusBadge } from "#/components/server-status-badge";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
  return (
    <main className="h-screen flex items-center justify-center gap-4 flex-col">
      <div className="flex items-center gap-4">
        <h1>Compositions</h1>
        <ModeToggle />
      </div>
      <ServerStatusBadge />
    </main>
  );
}

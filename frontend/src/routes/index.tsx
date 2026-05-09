import { ModeToggle } from "#/components/mode-toggle";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
  return (
    <div className="flex gap-4 items-center justify-center h-screen">
      <h1>Compositions</h1>
      <ModeToggle />
    </div>
  );
}

import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
  return (
    <div className="flex items-center justify-center h-screen">
      <h1>Compositions</h1>
    </div>
  );
}

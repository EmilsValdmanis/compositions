import { GameWebSocketActions } from "#/components/game-websocket-actions";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_protected/")({
  component: Home,
});

function Home() {
  return <GameWebSocketActions />;
}

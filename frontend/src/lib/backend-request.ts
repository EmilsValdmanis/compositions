import { getRequestHeaders } from "@tanstack/react-start/server";
import { backendURL } from "./backend-url";

export function requestBackend(
  path: string,
  {
    forwardSession = false,
    method = "GET",
  }: { forwardSession?: boolean; method?: "GET" | "POST" } = {},
) {
  const headers = new Headers({ accept: "application/json" });
  if (forwardSession) {
    const cookie = getRequestHeaders().get("cookie");
    if (cookie) headers.set("cookie", cookie);
  }

  return fetch(backendURL(path, process.env.VITE_GAME_SERVER_URL), { headers, method });
}

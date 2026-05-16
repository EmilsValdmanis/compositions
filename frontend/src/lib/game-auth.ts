import { auth } from "#/lib/auth";
import { resolveGameConnectionAuth } from "#/lib/game-connection-auth";
import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders, setResponseHeader } from "@tanstack/react-start/server";

export const getGameConnectionAuth = createServerFn({ method: "POST" }).handler(async () => {
  setResponseHeader("cache-control", "no-store");

  const headers = new Headers(getRequestHeaders());
  const session = await auth.api.getSession({ headers });

  return resolveGameConnectionAuth(headers, session);
});

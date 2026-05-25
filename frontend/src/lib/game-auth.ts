import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders, setResponseHeader } from "@tanstack/react-start/server";
import { auth } from "#/lib/auth";
import { resolveGameConnectionAuth } from "#/lib/game-connection-auth";
import { loadVerifiedSession } from "#/lib/verified-session";

export const getGameConnectionAuth = createServerFn({ method: "POST" }).handler(async () => {
  setResponseHeader("cache-control", "no-store");

  const headers = new Headers(getRequestHeaders());
  const session = await loadVerifiedSession(headers, auth.api.getSession);

  return resolveGameConnectionAuth(headers, session);
});

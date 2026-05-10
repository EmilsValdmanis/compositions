import { AUTH_COOKIE_PREFIX, AUTH_SESSION_COOKIE_NAME, auth } from "#/lib/auth";
import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders } from "@tanstack/react-start/server";
import { getSessionCookie } from "better-auth/cookies";

export const getGameConnectionAuth = createServerFn({ method: "GET" }).handler(async () => {
  const headers = new Headers(getRequestHeaders());
  const session = await auth.api.getSession({ headers });

  if (!session) return null;

  const authToken = getSessionCookie(headers, {
    cookiePrefix: AUTH_COOKIE_PREFIX,
    cookieName: AUTH_SESSION_COOKIE_NAME,
  });

  if (!authToken) return null;

  return {
    authToken,
    user: {
      id: session.user.id,
      name: session.user.name,
      email: session.user.email,
    },
  };
});

import { getSessionCookie } from "better-auth/cookies";

import { AUTH_COOKIE_PREFIX, AUTH_SESSION_COOKIE_NAME } from "#/lib/auth";

type GameConnectionSession = {
  user: {
    id: string;
    name: string;
    email: string;
  };
};

export type GameConnectionAuth = {
  authToken: string;
  user: {
    id: string;
    name: string;
    email: string;
  };
};

export function resolveGameConnectionAuth(
  headers: Headers,
  session: GameConnectionSession | null,
): GameConnectionAuth | null {
  if (!session) {
    return null;
  }

  const authToken = getSessionCookie(headers, {
    cookiePrefix: AUTH_COOKIE_PREFIX,
    cookieName: AUTH_SESSION_COOKIE_NAME,
  });

  if (!authToken) {
    return null;
  }

  return {
    authToken,
    user: {
      id: session.user.id,
      name: session.user.name,
      email: session.user.email,
    },
  };
}
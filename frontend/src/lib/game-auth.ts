import { auth } from "#/lib/auth";
import { createServerFn } from "@tanstack/react-start";
import { getCookie, getRequestHeaders } from "@tanstack/react-start/server";

const SESSION_COOKIE_NAME = "better-auth.session_token";

export const getGameConnectionAuth = createServerFn({ method: "GET" }).handler(async () => {
  const headers = new Headers(getRequestHeaders());
  const session = await auth.api.getSession({ headers });

  if (!session) return null;

  const authToken = getCookie(SESSION_COOKIE_NAME);

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

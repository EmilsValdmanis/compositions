import { createServerFn } from "@tanstack/react-start";
import { getRequestHeaders } from "@tanstack/react-start/server";
import { z } from "zod";
import { loadVerifiedSession } from "#/lib/verified-session";

export type AuthSession = {
  user: {
    id: string;
    name: string;
    email: string;
    image: string;
  };
};

function authURL(path: string) {
  const serverURL = process.env.VITE_GAME_SERVER_URL;

  if (!serverURL) {
    throw new Error("missing VITE_GAME_SERVER_URL");
  }

  return new URL(path, serverURL).toString();
}

async function getSessionFromBackend({ headers }: { headers: Headers }) {
  const cookie = headers.get("cookie");
  const response = await fetch(authURL("/auth/session"), {
    headers: cookie
      ? {
          cookie,
          accept: "application/json",
        }
      : {
          accept: "application/json",
        },
  });

  if (!response.ok) {
    throw new Error(`failed to load session: ${response.status}`);
  }

  return (await response.json()) as AuthSession | null;
}

const getSession = createServerFn({ method: "GET" })
  .inputValidator(z.undefined())
  .handler(async () => {
    const headers = new Headers(getRequestHeaders());
    return loadVerifiedSession(headers, ({ headers: requestHeaders }) =>
      getSessionFromBackend({ headers: requestHeaders }),
    );
  });

export const auth = {
  api: {
    getSession: ({ headers }: { headers: Headers }) => getSessionFromBackend({ headers }),
  },
};

export { getSession };

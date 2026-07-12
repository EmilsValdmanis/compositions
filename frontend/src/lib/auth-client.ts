import { authURL, type AuthSession } from "#/lib/auth-shared";

async function readSession() {
  const response = await fetch(authURL("/auth/session", import.meta.env.VITE_GAME_SERVER_URL), {
    credentials: "include",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`failed to load session: ${response.status}`);
  }

  return (await response.json()) as AuthSession | null;
}

async function signOut() {
  const response = await fetch(authURL("/auth/logout", import.meta.env.VITE_GAME_SERVER_URL), {
    method: "POST",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`failed to sign out: ${response.status}`);
  }
}

function signInWithGoogle(returnTo?: string) {
  const url = new URL(authURL("/auth/google", import.meta.env.VITE_GAME_SERVER_URL));
  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }
  window.location.assign(url);
}

export const authClient = {
  getSession: readSession,
  signOut,
  signIn: {
    social: async ({ provider, returnTo }: { provider: string; returnTo?: string }) => {
      if (provider !== "google") {
        throw new Error(`unsupported provider: ${provider}`);
      }

      signInWithGoogle(returnTo);
    },
  },
};

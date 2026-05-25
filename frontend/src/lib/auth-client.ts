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

function signInWithGoogle() {
  window.location.assign(authURL("/auth/google", import.meta.env.VITE_GAME_SERVER_URL));
}

export const authClient = {
  getSession: readSession,
  signOut,
  signIn: {
    social: async ({ provider }: { provider: string }) => {
      if (provider !== "google") {
        throw new Error(`unsupported provider: ${provider}`);
      }

      signInWithGoogle();
    },
  },
};

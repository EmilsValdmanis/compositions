export type AuthSession = {
  user: {
    id: string;
    name: string;
    email: string;
    image: string;
  };
};

function authURL(path: string) {
  const serverURL = import.meta.env.VITE_GAME_SERVER_URL;

  if (!serverURL) {
    throw new Error("missing VITE_GAME_SERVER_URL");
  }

  return new URL(path, serverURL).toString();
}

async function readSession() {
  const response = await fetch(authURL("/auth/session"), {
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
  const response = await fetch(authURL("/auth/logout"), {
    method: "POST",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`failed to sign out: ${response.status}`);
  }
}

function signInWithGoogle() {
  window.location.assign(authURL("/auth/google"));
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

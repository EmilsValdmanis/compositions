import { backendURL } from "./backend-url";

async function postAuth(path: string, action: string) {
  const response = await fetch(backendURL(path, import.meta.env.VITE_GAME_SERVER_URL), {
    method: "POST",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`failed to ${action}: ${response.status}`);
  }
}

function signInWithGoogle(returnTo?: string) {
  const url = new URL(backendURL("/auth/google", import.meta.env.VITE_GAME_SERVER_URL));
  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }
  window.location.assign(url);
}

export const authClient = {
  completeOnboarding: () => postAuth("/auth/onboarding/complete", "complete onboarding"),
  signOut: () => postAuth("/auth/logout", "sign out"),
  signInWithGoogle,
};

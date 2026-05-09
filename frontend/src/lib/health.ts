import { createServerFn } from "@tanstack/react-start";

export const checkGameServerHealth = createServerFn({ method: "GET" }).handler(async () => {
  const gameServerUrl = import.meta.env.VITE_GAME_SERVER_URL;

  try {
    const url = new URL("/health", gameServerUrl);
    const response = await fetch(url.toString(), {
      headers: {
        Accept: "application/json",
      },
    });
    return { status: response.ok ? "online" : "offline" } as const;
  } catch {
    return { status: "offline" } as const;
  }
});

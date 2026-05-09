import { createServerFn } from "@tanstack/react-start";

export const checkGameServerHealth = createServerFn({ method: "GET" }).handler(async () => {
  const serverUrl = process.env.GAME_SERVER_URL;

  try {
    const url = new URL("/health", serverUrl);
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

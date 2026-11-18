export function backendURL(path: string, serverURL: string | undefined) {
  if (!serverURL) {
    throw new Error("missing VITE_GAME_SERVER_URL");
  }

  return new URL(path, serverURL).toString();
}

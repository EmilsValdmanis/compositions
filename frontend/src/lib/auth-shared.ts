export type AuthSession = {
  user: {
    id: string;
    name: string;
    email: string;
    image: string;
  };
};

export function authURL(path: string, serverURL: string | undefined) {
  if (!serverURL) {
    throw new Error("missing VITE_GAME_SERVER_URL");
  }

  return new URL(path, serverURL).toString();
}

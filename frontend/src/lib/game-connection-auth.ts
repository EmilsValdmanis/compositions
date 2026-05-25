type GameConnectionSession = {
  user: {
    id: string;
    name: string;
    email: string;
  };
};

export type GameConnectionAuth = {
  user: {
    id: string;
    name: string;
    email: string;
  };
};

export function resolveGameConnectionAuth(
  _headers: Headers,
  session: GameConnectionSession | null,
): GameConnectionAuth | null {
  if (!session) {
    return null;
  }

  return {
    user: {
      id: session.user.id,
      name: session.user.name,
      email: session.user.email,
    },
  };
}

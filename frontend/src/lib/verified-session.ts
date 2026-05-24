const verifiedSessionQuery = {
  disableCookieCache: true,
} as const;

type VerifiedSessionLoader<Session> = (input: {
  headers: Headers;
  query: typeof verifiedSessionQuery;
}) => Promise<Session | null>;

export function loadVerifiedSession<Session>(
  headers: Headers,
  getSession: VerifiedSessionLoader<Session>,
) {
  return getSession({
    headers,
    query: verifiedSessionQuery,
  });
}

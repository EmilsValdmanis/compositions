export const SIDEBAR_STATE_COOKIE = "sidebar_state";
const SIDEBAR_STATE_MAX_AGE = 7 * 24 * 60 * 60;

export function isSidebarOpen(cookie: string | undefined) {
  return cookie === "true";
}

export function sidebarStateCookie(open: boolean) {
  return `${SIDEBAR_STATE_COOKIE}=${open}; path=/; max-age=${SIDEBAR_STATE_MAX_AGE}`;
}

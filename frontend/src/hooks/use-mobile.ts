import { useSyncExternalStore } from "react";

const MOBILE_QUERY = "(width < 48rem)";

function subscribe(onChange: () => void) {
  const media = window.matchMedia?.(MOBILE_QUERY);
  media?.addEventListener("change", onChange);
  return () => media?.removeEventListener("change", onChange);
}

function readIsMobile() {
  return window.matchMedia?.(MOBILE_QUERY).matches ?? false;
}

export function useIsMobile() {
  return useSyncExternalStore(subscribe, readIsMobile, () => false);
}

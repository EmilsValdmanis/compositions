import { useSyncExternalStore } from "react";

const REDUCED_MOTION_PREFERENCE_KEY = "compositions.reduce-motion-enabled";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";
const preferenceListeners = new Set<() => void>();

export function isReducedMotionPreferenceEnabled() {
  if (typeof window === "undefined") {
    return false;
  }

  return window.localStorage.getItem(REDUCED_MOTION_PREFERENCE_KEY) === "true";
}

export function shouldReduceMotion() {
  if (typeof window === "undefined") {
    return false;
  }

  return (
    isReducedMotionPreferenceEnabled() || window.matchMedia?.(REDUCED_MOTION_QUERY).matches === true
  );
}

export function subscribeToReducedMotionPreference(listener: () => void) {
  preferenceListeners.add(listener);

  if (typeof window === "undefined") {
    return () => preferenceListeners.delete(listener);
  }

  const mediaQuery = window.matchMedia?.(REDUCED_MOTION_QUERY);
  const handleStorage = (event: StorageEvent) => {
    if (event.key === REDUCED_MOTION_PREFERENCE_KEY) {
      listener();
    }
  };

  mediaQuery?.addEventListener("change", listener);
  window.addEventListener("storage", handleStorage);

  return () => {
    preferenceListeners.delete(listener);
    mediaQuery?.removeEventListener("change", listener);
    window.removeEventListener("storage", handleStorage);
  };
}

function applyReducedMotionPreference() {
  if (typeof document === "undefined") {
    return;
  }

  document.documentElement.toggleAttribute(
    "data-reduce-motion",
    isReducedMotionPreferenceEnabled(),
  );
}

export function setReducedMotionPreferenceEnabled(enabled: boolean) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(REDUCED_MOTION_PREFERENCE_KEY, String(enabled));
  }

  applyReducedMotionPreference();

  for (const listener of preferenceListeners) {
    listener();
  }
}

export function useReducedMotionPreference() {
  return useSyncExternalStore(
    subscribeToReducedMotionPreference,
    isReducedMotionPreferenceEnabled,
    () => false,
  );
}

export function useShouldReduceMotion() {
  return useSyncExternalStore(subscribeToReducedMotionPreference, shouldReduceMotion, () => false);
}

export function getReducedMotionScript() {
  return "(function(){try{if(localStorage.getItem('compositions.reduce-motion-enabled')==='true'){document.documentElement.setAttribute('data-reduce-motion','')}}catch(e){}})();";
}

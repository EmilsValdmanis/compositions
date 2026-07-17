import { useSyncExternalStore } from "react";
import { Toaster } from "#/components/ui/sonner";
import { useTheme } from "#/components/theme-provider";

const MOBILE_TOASTER_QUERY = "(max-width: 47.999rem)";

function subscribeToMobileLayout(onChange: () => void) {
  if (typeof window === "undefined") return () => {};

  const mediaQuery = window.matchMedia(MOBILE_TOASTER_QUERY);
  mediaQuery.addEventListener("change", onChange);
  return () => mediaQuery.removeEventListener("change", onChange);
}

function isMobileLayout() {
  return typeof window !== "undefined" && window.matchMedia(MOBILE_TOASTER_QUERY).matches;
}

export function ThemeAwareToaster() {
  const { theme } = useTheme();
  const isMobile = useSyncExternalStore(subscribeToMobileLayout, isMobileLayout, () => false);

  return (
    <Toaster
      richColors
      theme={theme}
      position={isMobile ? "top-center" : "bottom-right"}
      mobileOffset={{
        top: "calc(env(safe-area-inset-top) + 0.75rem)",
        right: "0.75rem",
        left: "0.75rem",
      }}
    />
  );
}

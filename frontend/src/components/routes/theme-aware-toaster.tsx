import { Toaster } from "#/components/ui/sonner";
import { useTheme } from "#/components/theme-provider";
import { useIsMobile } from "#/hooks/use-mobile";

export function ThemeAwareToaster() {
  const { theme } = useTheme();
  const isMobile = useIsMobile();

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

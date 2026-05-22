import { Toaster } from "sonner";
import { useTheme } from "#/components/theme-provider";

export function ThemeAwareToaster() {
  const { theme } = useTheme();

  return <Toaster richColors theme={theme} />;
}

import { ComputerIcon, Moon02Icon, Sun03Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useTheme } from "#/components/theme-provider";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

const themeOptions = [
  { value: "light", label: m.theme_light, icon: Sun03Icon },
  { value: "dark", label: m.theme_dark, icon: Moon02Icon },
  { value: "system", label: m.theme_system, icon: ComputerIcon },
] as const;

export function ThemeSwitcher({
  className,
  align = "end",
}: {
  className?: string;
  align?: "start" | "center" | "end";
}) {
  const { theme, setTheme } = useTheme();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className={cn("relative", className)}
            aria-label={m.theme()}
          />
        }
      >
        <HugeiconsIcon
          icon={Sun03Icon}
          data-slot="theme-light-icon"
          className="transition-[transform,opacity] dark:-rotate-90 dark:scale-95 dark:opacity-0"
        />
        <HugeiconsIcon
          icon={Moon02Icon}
          data-slot="theme-dark-icon"
          className="absolute rotate-90 scale-95 opacity-0 transition-[transform,opacity] dark:rotate-0 dark:scale-100 dark:opacity-100"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-44">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{m.theme()}</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={theme}
            onValueChange={(value) => setTheme(value as typeof theme)}
          >
            {themeOptions.map((option) => (
              <DropdownMenuRadioItem key={option.value} value={option.value}>
                <HugeiconsIcon icon={option.icon} />
                {option.label()}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

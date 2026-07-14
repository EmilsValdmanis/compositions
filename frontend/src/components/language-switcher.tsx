import { Globe02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { m } from "#/paraglide/messages.js";
import { getLocale, locales, setLocale, type Locale } from "#/paraglide/runtime.js";

const localeLabels: Record<Locale, () => string> = {
  en: m.language_english,
  lv: m.language_latvian,
};

function LanguageOptions() {
  const currentLocale = getLocale();

  return (
    <DropdownMenuRadioGroup
      value={currentLocale}
      onValueChange={(locale) => setLocale(locale as Locale)}
    >
      {locales.map((locale) => (
        <DropdownMenuRadioItem key={locale} value={locale}>
          {localeLabels[locale]()}
        </DropdownMenuRadioItem>
      ))}
    </DropdownMenuRadioGroup>
  );
}

export function LanguageSubmenu() {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <HugeiconsIcon icon={Globe02Icon} />
        {m.language()}
      </DropdownMenuSubTrigger>
      <DropdownMenuSubContent>
        <DropdownMenuGroup>
          <LanguageOptions />
        </DropdownMenuGroup>
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );
}

export function LanguageSwitcher({ className }: { className?: string }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className={className}
            aria-label={m.language()}
          />
        }
      >
        <HugeiconsIcon icon={Globe02Icon} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{m.language()}</DropdownMenuLabel>
          <LanguageOptions />
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

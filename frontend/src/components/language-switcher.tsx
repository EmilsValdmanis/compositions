import { Globe02Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Button } from "#/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { m } from "#/paraglide/messages.js";
import { getLocale, locales, setLocale, type Locale } from "#/paraglide/runtime.js";

const localeLabels: Record<Locale, () => string> = {
  en: m.language_english,
  lv: m.language_latvian,
};

export function LanguageSwitcher({ className }: { className?: string }) {
  const currentLocale = getLocale();

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
          {locales.map((locale) => (
            <DropdownMenuItem key={locale} onClick={() => setLocale(locale)}>
              <span>{localeLabels[locale]()}</span>
              {locale === currentLocale ? (
                <HugeiconsIcon icon={Tick02Icon} className="ml-auto" />
              ) : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

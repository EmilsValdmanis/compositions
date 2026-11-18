import { createContext, use, useEffect, useState } from "react";
import { ScriptOnce } from "@tanstack/react-router";

const THEMES = ["light", "dark", "system"] as const;
type Theme = (typeof THEMES)[number];

type ThemeProviderProps = {
  children: React.ReactNode;
};

const DEFAULT_THEME: Theme = "system";
const THEME_STORAGE_KEY = "theme";
const DARK_MODE_QUERY = "(prefers-color-scheme: dark)";
const THEME_SCRIPT = `
  (() => {
    let theme = ${JSON.stringify(DEFAULT_THEME)};
    try {
      const stored = localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});
      if (${JSON.stringify(THEMES)}.includes(stored)) theme = stored;
    } catch {}
    const resolved = theme === "system" ? (matchMedia(${JSON.stringify(DARK_MODE_QUERY)}).matches ? "dark" : "light") : theme;
    const root = document.documentElement;
    root.classList.add(resolved);
    root.style.colorScheme = resolved;
  })();
`;

type ThemeProviderState = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
};

function isTheme(value: string | null): value is Theme {
  return THEMES.some((theme) => theme === value);
}

function readTheme(): Theme {
  if (typeof window !== "undefined") {
    try {
      const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
      if (isTheme(stored)) return stored;
    } catch {
      // Some browsers disable storage in restricted contexts.
    }
  }
  return DEFAULT_THEME;
}

const ThemeProviderContext = createContext<ThemeProviderState | null>(null);

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.classList.remove("light", "dark");

  const resolved =
    theme === "system" ? (window.matchMedia(DARK_MODE_QUERY).matches ? "dark" : "light") : theme;

  root.classList.add(resolved);
  root.style.colorScheme = resolved;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const [theme, setThemeState] = useState(readTheme);

  useEffect(() => {
    applyTheme(theme);

    if (theme !== "system") {
      return;
    }

    const media = window.matchMedia(DARK_MODE_QUERY);
    const onChange = () => applyTheme("system");
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [theme]);

  const setTheme = (next: Theme) => {
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // The theme can still change for the current page when storage is blocked.
    }
    setThemeState(next);
  };

  return (
    <ThemeProviderContext value={{ theme, setTheme }}>
      <ScriptOnce>{THEME_SCRIPT}</ScriptOnce>
      {children}
    </ThemeProviderContext>
  );
}

export function useTheme() {
  const context = use(ThemeProviderContext);
  if (!context) throw new Error("useTheme must be used within a ThemeProvider");
  return context;
}

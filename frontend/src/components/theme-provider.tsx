import { createContext, use, useEffect, useState } from "react";
import { ScriptOnce } from "@tanstack/react-router";

type Theme = "dark" | "light" | "system";

type ThemeProviderProps = {
  children: React.ReactNode;
};

const DEFAULT_THEME: Theme = "system";
const THEME_STORAGE_KEY = "theme";
const THEME_SCRIPT =
  "(function(){try{var t=localStorage.getItem('theme');if(t!=='light'&&t!=='dark'&&t!=='system'){t='system'}var d=matchMedia('(prefers-color-scheme: dark)').matches;var r=t==='system'?(d?'dark':'light'):t;var e=document.documentElement;e.classList.add(r);e.style.colorScheme=r}catch(e){}})();";

type ThemeProviderState = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
};

function resolveTheme(storageKey: string, fallbackTheme: Theme): Theme {
  if (typeof window === "undefined") {
    return fallbackTheme;
  }

  const stored = localStorage.getItem(storageKey);
  return stored === "light" || stored === "dark" || stored === "system" ? stored : fallbackTheme;
}

const ThemeProviderContext = createContext<ThemeProviderState | null>(null);

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.classList.remove("light", "dark");

  const resolved =
    theme === "system"
      ? window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : theme;

  root.classList.add(resolved);
  root.style.colorScheme = resolved;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(() =>
    resolveTheme(THEME_STORAGE_KEY, DEFAULT_THEME),
  );

  useEffect(() => {
    applyTheme(theme);

    if (theme !== "system") {
      return;
    }

    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyTheme("system");
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [theme]);

  const setTheme = (next: Theme) => {
    localStorage.setItem(THEME_STORAGE_KEY, next);
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

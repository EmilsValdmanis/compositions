// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";

const setTheme = vi.fn();

vi.mock("#/components/theme-provider", () => ({
  useTheme: () => ({ theme: "system", setTheme }),
}));

const { ThemeSwitcher } = await import("#/components/theme-switcher");

beforeEach(() => setTheme.mockClear());
afterEach(cleanup);

describe("ThemeSwitcher", () => {
  it("opens from its icon button and changes the selected theme", async () => {
    render(<ThemeSwitcher className="absolute top-4 left-4" align="start" />);

    const trigger = screen.getByRole("button", { name: "Theme" });
    expect(trigger.className).toContain("left-4");
    expect(trigger.querySelector('[data-slot="theme-light-icon"]')).toBeTruthy();
    expect(trigger.querySelector('[data-slot="theme-dark-icon"]')).toBeTruthy();

    fireEvent.click(trigger);
    fireEvent.click(await screen.findByRole("menuitemradio", { name: "Dark" }));

    expect(setTheme).toHaveBeenCalledWith("dark");
  });
});

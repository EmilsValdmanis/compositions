// @vitest-environment jsdom
import { createRequire } from "node:module";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  isReducedMotionPreferenceEnabled,
  setReducedMotionPreferenceEnabled,
  shouldReduceMotion,
  subscribeToReducedMotionPreference,
} from "#/lib/reduced-motion";

if (typeof globalThis.document === "undefined") {
  const require = createRequire(import.meta.url);
  const { JSDOM } = require("jsdom") as {
    JSDOM: new (
      html?: string,
      options?: Record<string, unknown>,
    ) => {
      window: Window & typeof globalThis;
    };
  };
  const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    url: "http://localhost/",
  });

  globalThis.window = dom.window;
  globalThis.document = dom.window.document;
}

function mockSystemReducedMotion(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockReturnValue({
      matches,
      media: "(prefers-reduced-motion: reduce)",
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  });
}

describe("reduced motion preference", () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute("data-reduce-motion");
    mockSystemReducedMotion(false);
  });

  it("stores and applies the in-app preference", () => {
    setReducedMotionPreferenceEnabled(true);

    expect(isReducedMotionPreferenceEnabled()).toBe(true);
    expect(shouldReduceMotion()).toBe(true);
    expect(document.documentElement.hasAttribute("data-reduce-motion")).toBe(true);

    setReducedMotionPreferenceEnabled(false);

    expect(isReducedMotionPreferenceEnabled()).toBe(false);
    expect(document.documentElement.hasAttribute("data-reduce-motion")).toBe(false);
  });

  it("continues to respect the operating-system preference", () => {
    mockSystemReducedMotion(true);

    expect(isReducedMotionPreferenceEnabled()).toBe(false);
    expect(shouldReduceMotion()).toBe(true);
  });

  it("notifies subscribers when the in-app preference changes", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToReducedMotionPreference(listener);

    setReducedMotionPreferenceEnabled(true);

    expect(listener).toHaveBeenCalledOnce();
    unsubscribe();
  });
});

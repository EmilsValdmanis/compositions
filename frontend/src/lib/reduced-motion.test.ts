// @vitest-environment jsdom
import { runInNewContext } from "node:vm";
import { beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  REDUCED_MOTION_SCRIPT,
  isReducedMotionPreferenceEnabled,
  setReducedMotionPreferenceEnabled,
  shouldReduceMotion,
  subscribeToReducedMotionPreference,
} from "#/lib/reduced-motion";

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

  it("restores the stored CSS preference before React renders", () => {
    window.localStorage.setItem("compositions.reduce-motion-enabled", "true");
    runInNewContext(REDUCED_MOTION_SCRIPT, { document, localStorage: window.localStorage });

    expect(document.documentElement.hasAttribute("data-reduce-motion")).toBe(true);
  });

  it("updates the CSS preference after another tab clears storage", () => {
    setReducedMotionPreferenceEnabled(true);
    const listener = vi.fn();
    const unsubscribe = subscribeToReducedMotionPreference(listener);
    try {
      window.localStorage.clear();
      window.dispatchEvent(new StorageEvent("storage", { key: null, storageArea: localStorage }));

      expect(document.documentElement.hasAttribute("data-reduce-motion")).toBe(false);
      expect(listener).toHaveBeenCalledOnce();
    } finally {
      unsubscribe();
    }
  });
});

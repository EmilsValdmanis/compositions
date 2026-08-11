// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";

const { fireConfettiMock } = vi.hoisted(() => ({
  fireConfettiMock: vi.fn(),
}));

vi.mock("canvas-confetti", () => ({
  default: fireConfettiMock,
}));

const { fireCelebrationConfetti, fireStreamingCelebrationConfetti } =
  await import("#/lib/confetti");

const LIGHT_THEME_COLORS = ["#0b2638", "#0d4d7d", "#176ca5", "#278bc7", "#55b6e8"];
const DARK_THEME_COLORS = ["#f4fbff", "#a5d8f3", "#55b6e8", "#278bc7", "#176ca5"];

describe("celebration confetti themes", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    fireConfettiMock.mockClear();
    document.documentElement.classList.remove("light", "dark");
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("uses dark contrasting flakes instead of white in light mode", async () => {
    document.documentElement.classList.add("light");

    await fireCelebrationConfetti();

    expect(fireConfettiMock).toHaveBeenCalledTimes(5);
    for (const [options] of fireConfettiMock.mock.calls) {
      expect(options.colors).toEqual(LIGHT_THEME_COLORS);
      expect(options.colors).not.toContain("#f4fbff");
    }
  });

  it("keeps bright flakes for contrast in dark mode", async () => {
    document.documentElement.classList.add("dark");

    await fireCelebrationConfetti();

    for (const [options] of fireConfettiMock.mock.calls) {
      expect(options.colors).toEqual(DARK_THEME_COLORS);
    }
  });

  it("uses the resolved theme for the side-streaming celebration", async () => {
    document.documentElement.classList.add("light");

    await fireStreamingCelebrationConfetti({ durationMs: 500 });

    expect(fireConfettiMock).toHaveBeenCalledTimes(2);
    for (const [options] of fireConfettiMock.mock.calls) {
      expect(options.colors).toEqual(LIGHT_THEME_COLORS);
    }

    await vi.advanceTimersByTimeAsync(500);
  });
});

// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { TurnStartCue } from "#/components/game/turn-start-cue";

afterEach(cleanup);

describe("TurnStartCue", () => {
  it("centers within its positioned game-board container without clipping its shadow", () => {
    render(<TurnStartCue round={4} turnNumber={9} playerName="Avery" />);

    const cue = screen.getByRole("status").closest('[data-slot="turn-start-cue"]');

    expect(cue?.className).toContain("absolute");
    expect(cue?.className).toContain("inset-0");
    expect(cue?.className).toContain("place-items-center");
    expect((cue as HTMLElement | null)?.style.clipPath).toBe("");
  });
});

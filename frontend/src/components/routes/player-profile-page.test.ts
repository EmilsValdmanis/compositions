// @vitest-environment jsdom

import { createElement, type ReactNode } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { PlayerProfile } from "#/lib/player-profile";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => createElement("a", { href: "/" }, children),
}));

const { PlayerProfilePage } = await import("./player-profile-page");

const profile: PlayerProfile = {
  id: "00000000-0000-0000-0000-000000000002",
  name: "Avery",
  imageUrl: "",
  gamesPlayed: 8,
  gamesWon: 3,
  totalPlacement: 14,
  totalPlaytimeSeconds: 5_430,
  roundsPlayed: 22,
  roundsWon: 9,
  compositionsCreated: 41,
  setsCreated: 17,
  runsCreated: 24,
  pointsInflicted: 230,
  penaltyPoints: 98,
  currentGameWinStreak: 1,
  longestGameWinStreak: 2,
  currentRoundWinStreak: 2,
  longestRoundWinStreak: 4,
};

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("PlayerProfilePage", () => {
  it("shows sharing on the signed-in player's own profile", () => {
    render(createElement(PlayerProfilePage, { profile, isOwnProfile: true }));

    expect(screen.getByRole("button", { name: "Share profile" })).toBeTruthy();
  });

  it("hides sharing on another player's profile", () => {
    render(createElement(PlayerProfilePage, { profile, isOwnProfile: false }));

    expect(screen.queryByRole("button", { name: "Share profile" })).toBeNull();
  });

  it("copies only the profile URL", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    render(createElement(PlayerProfilePage, { profile, isOwnProfile: true }));

    fireEvent.click(screen.getByRole("button", { name: "Share profile" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith(window.location.href));
  });
});

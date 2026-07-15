// @vitest-environment jsdom

import { createElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  playerGameHistorySchema,
  type PlayerGameHistory,
  type PlayerProfile,
} from "#/lib/player-profile";

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

const history: PlayerGameHistory = {
  games: [
    {
      id: "123e4567-e89b-42d3-a456-426614174000",
      status: "completed",
      completedAt: "2026-07-15T12:30:00.000Z",
      placement: 1,
      playerCount: 3,
      won: true,
      forfeited: false,
      totalPoints: 42,
      roundsPlayed: 4,
      roundsWon: 2,
      playtimeSeconds: 1_800,
    },
  ],
  page: 1,
  pageSize: 10,
  totalItems: 1,
  totalPages: 1,
};

function renderProfile(isOwnProfile: boolean) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(PlayerProfilePage, { profile, initialHistory: history, isOwnProfile }),
    ),
  );
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("PlayerProfilePage", () => {
  it("accepts game timestamps returned with a database timezone offset", () => {
    const offsetHistory = {
      ...history,
      games: [{ ...history.games[0], completedAt: "2026-07-15T12:30:00+03:00" }],
    };

    expect(playerGameHistorySchema.parse(offsetHistory).games[0]?.completedAt).toBe(
      "2026-07-15T12:30:00+03:00",
    );
  });

  it("shows sharing on the signed-in player's own profile", () => {
    renderProfile(true);

    expect(screen.getByRole("button", { name: "Share profile" })).toBeTruthy();
  });

  it("hides sharing on another player's profile", () => {
    renderProfile(false);

    expect(screen.queryByRole("button", { name: "Share profile" })).toBeNull();
  });

  it("copies only the profile URL", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    renderProfile(true);

    fireEvent.click(screen.getByRole("button", { name: "Share profile" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith(window.location.href));
  });

  it("shows the first server-paginated game-history page", () => {
    renderProfile(false);

    expect(screen.getByRole("table", { name: "Game history" })).toBeTruthy();
    expect(screen.getByText("Victory")).toBeTruthy();
    expect(screen.getByText("42")).toBeTruthy();
    expect(screen.getByText("Page 1 of 1")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Previous" })).toHaveProperty("disabled", true);
    expect(screen.getByRole("button", { name: "Next" })).toHaveProperty("disabled", true);
  });
});

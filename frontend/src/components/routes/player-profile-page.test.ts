// @vitest-environment jsdom

import { createElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
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
  rankedFull: {
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
  },
  quick: {
    gamesPlayed: 2,
    gamesWon: 1,
    totalPlacement: 3,
    totalPlaytimeSeconds: 900,
    roundsPlayed: 2,
    roundsWon: 1,
    compositionsCreated: 8,
    setsCreated: 3,
    runsCreated: 5,
    pointsInflicted: 40,
    penaltyPoints: 20,
    currentGameWinStreak: 1,
    longestGameWinStreak: 1,
    currentRoundWinStreak: 1,
    longestRoundWinStreak: 1,
  },
};

const history: PlayerGameHistory = {
  games: [
    {
      id: "123e4567-e89b-42d3-a456-426614174000",
      status: "completed",
      gameMode: "full",
      ranked: true,
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

function renderProfile(initialHistory = history) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(PlayerProfilePage, { profile, initialHistory }),
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

  it("shows game history without pagination when there is only one page", () => {
    renderProfile();

    expect(screen.getByRole("table", { name: "Game history" })).toBeTruthy();
    expect(screen.getByText("Victory")).toBeTruthy();
    expect(screen.getByText("42")).toBeTruthy();
    expect(screen.queryByText("Page 1 of 1")).toBeNull();
    expect(screen.queryByRole("button", { name: "Previous" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next" })).toBeNull();
  });

  it("shows the empty component when game history has no games", () => {
    renderProfile({
      games: [],
      page: 1,
      pageSize: 10,
      totalItems: 0,
      totalPages: 0,
    });

    expect(screen.getByText("No games yet")).toBeTruthy();
    expect(screen.getByText("Completed games will appear here.")).toBeTruthy();
    expect(screen.queryByRole("table", { name: "Game history" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Previous" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next" })).toBeNull();
  });

  it("shows pagination when game history has multiple pages", () => {
    renderProfile({ ...history, totalItems: 11, totalPages: 2 });

    expect(screen.getByText("Page 1 of 2")).toBeTruthy();
    const previous = screen.getByRole("button", { name: "Previous" });
    const next = screen.getByRole("button", { name: "Next" });
    expect(previous.getAttribute("aria-disabled")).toBe("true");
    expect(previous).toHaveProperty("tabIndex", -1);
    expect(next.getAttribute("aria-disabled")).toBe("false");
    expect(next).toHaveProperty("tabIndex", 0);
  });

  it("uses one compact mode toggle for the whole profile", () => {
    renderProfile();

    expect(screen.getByRole("button", { name: "All" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Ranked" })).toBeTruthy();
    const quickButton = screen.getByRole("button", { name: "Quick" });
    fireEvent.click(quickButton);

    expect(screen.getByText("Quick record")).toBeTruthy();
    expect(screen.getByText("Casual quick finishes")).toBeTruthy();
  });
});

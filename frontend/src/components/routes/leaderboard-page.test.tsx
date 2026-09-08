// @vitest-environment jsdom
import type { ReactNode } from "react";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { getLocale, overwriteGetLocale } from "#/paraglide/runtime.js";

const { useInfiniteQuery } = vi.hoisted(() => ({ useInfiniteQuery: vi.fn() }));
vi.mock("@tanstack/react-query", () => ({
  useInfiniteQuery,
  infiniteQueryOptions: (options: unknown) => options,
}));
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => <a>{children}</a>,
}));
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: () => ({
    getVirtualItems: () => [{ index: 0, key: "player", start: 0, end: 57 }],
    getTotalSize: () => 57,
    measureElement: () => {},
  }),
}));
const { LeaderboardPage } = await import("./leaderboard-page");
const originalGetLocale = getLocale;
afterEach(() => {
  cleanup();
  overwriteGetLocale(originalGetLocale);
  vi.clearAllMocks();
});

function showPlayers(tier = "gold") {
  useInfiniteQuery.mockReturnValue({
    data: {
      pages: [
        {
          players: [{ playerId: "player", name: "Avery", rank: 1, score: 1450, tier }],
          placement: { playerId: "viewer", name: "Casey", rank: 22, score: 980, tier: "bronze" },
        },
      ],
    },
  });
}

it("opens Elo as the first/default tab with rating, tier and pinned placement", () => {
  overwriteGetLocale(() => "en");
  showPlayers();
  render(<LeaderboardPage playerId="viewer" />);
  const tabs = screen.getAllByRole("tab");
  expect(tabs[0].textContent).toBe("Elo rating");
  expect(tabs[0].getAttribute("aria-selected")).toBe("true");
  expect(useInfiniteQuery.mock.lastCall?.[0].queryKey).toEqual([
    "leaderboard",
    "viewer",
    "elo",
    "friends",
  ]);
  expect(screen.getByText("Gold")).toBeTruthy();
  expect(screen.getByText("1,450")).toBeTruthy();
  const placement = screen.getByRole("row", { name: "Your placement" });
  expect(within(placement).getByText("Bronze")).toBeTruthy();
  expect(within(placement).getByText("980")).toBeTruthy();
});

it("switches statistics and scope without showing a tier for a wins score", () => {
  overwriteGetLocale(() => "en");
  showPlayers();
  render(<LeaderboardPage playerId="viewer" />);
  // jsdom does not implement Element.scrollTo.
  Element.prototype.scrollTo = vi.fn();
  fireEvent.click(screen.getByRole("tab", { name: "Wins" }));
  expect(useInfiniteQuery.mock.lastCall?.[0].queryKey).toEqual([
    "leaderboard",
    "viewer",
    "wins",
    "friends",
  ]);
  expect(screen.queryByText("Gold")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Global" }));
  expect(useInfiniteQuery.mock.lastCall?.[0].queryKey).toEqual([
    "leaderboard",
    "viewer",
    "wins",
    "global",
  ]);
  fireEvent.click(screen.getByRole("tab", { name: "Elo rating" }));
  expect(screen.getByText("Gold")).toBeTruthy();
});

it.each([
  ["bronze", "Bronze", "Bronza"],
  ["silver", "Silver", "Sudrabs"],
  ["gold", "Gold", "Zelts"],
  ["platinum", "Platinum", "Platīns"],
  ["diamond", "Diamond", "Dimants"],
  ["master", "Master", "Meistars"],
  ["grandmaster", "Grand Master", "Lielmeistars"],
])("translates %s in both supported languages", (tier, en, lv) => {
  for (const locale of ["en", "lv"] as const) {
    overwriteGetLocale(() => locale);
    showPlayers(tier);
    render(<LeaderboardPage playerId="viewer" />);
    expect(screen.getAllByText(locale === "en" ? en : lv).length).toBeGreaterThan(0);
    cleanup();
  }
});

it("keeps Elo selected for an empty leaderboard", () => {
  overwriteGetLocale(() => "en");
  useInfiniteQuery.mockReturnValue({ data: { pages: [{ players: [], placement: null }] } });
  render(<LeaderboardPage playerId="viewer" />);
  expect(screen.getByRole("tab", { name: "Elo rating" }).getAttribute("aria-selected")).toBe(
    "true",
  );
  expect(screen.queryByRole("table")).toBeNull();
});

it("renders a future server tier without breaking the leaderboard", () => {
  overwriteGetLocale(() => "en");
  showPlayers("champion");
  render(<LeaderboardPage playerId="viewer" />);
  expect(screen.getByText("champion")).toBeTruthy();
  expect(screen.getByText("1,450")).toBeTruthy();
});

it("replaces the visible ranking and pinned placement after a stale cursor reset", () => {
  overwriteGetLocale(() => "en");
  const scrollTo = vi.fn();
  Element.prototype.scrollTo = scrollTo;
  const oldPage = {
    players: [{ playerId: "A", name: "Old leader", rank: 1, score: 1016, tier: "bronze" }],
    placement: { playerId: "viewer", name: "Viewer", rank: 3, score: 984, tier: "bronze" },
  };
  useInfiniteQuery.mockReturnValue({ data: { pages: [oldPage] } });
  const view = render(<LeaderboardPage playerId="viewer" />);
  const freshPage = {
    reset: true,
    players: [{ playerId: "C", name: "New leader", rank: 1, score: 1001, tier: "bronze" }],
    placement: { playerId: "viewer", name: "Viewer", rank: 2, score: 1000, tier: "bronze" },
  };
  useInfiniteQuery.mockReturnValue({ data: { pages: [oldPage, freshPage] } });
  view.rerender(<LeaderboardPage playerId="viewer" />);
  expect(screen.queryByText("Old leader")).toBeNull();
  expect(screen.getByText("New leader")).toBeTruthy();
  expect(
    within(screen.getByRole("row", { name: "Your placement" })).getByText("1,000"),
  ).toBeTruthy();
  expect(scrollTo).toHaveBeenCalledWith({ top: 0 });
});

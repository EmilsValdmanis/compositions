// @vitest-environment jsdom

import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  type GameSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";

vi.mock("#/components/ui/animated-number", () => ({
  AnimatedNumber: ({
    value,
    prefix,
    trend,
  }: {
    value: number;
    prefix?: string;
    trend?: number;
  }) => (
    <span data-number-trend={trend}>
      {prefix}
      {value}
    </span>
  ),
}));

vi.mock("#/components/game/game-board-players", () => ({
  GameBoardPlayers: () => null,
}));

const { fireCelebrationConfettiMock, fireStreamingCelebrationConfettiMock } = vi.hoisted(() => ({
  fireCelebrationConfettiMock: vi.fn(),
  fireStreamingCelebrationConfettiMock: vi.fn(),
}));

vi.mock("#/lib/confetti", () => ({
  fireCelebrationConfetti: fireCelebrationConfettiMock,
  fireStreamingCelebrationConfetti: fireStreamingCelebrationConfettiMock,
}));

const { GameResultsView } = await import("#/components/game/game-results-view");

const players: PlayerSnapshot[] = [
  {
    playerId: "winner",
    name: "Avery",
    connected: true,
    seat: 0,
    isHost: true,
    canReconnect: true,
  },
  {
    playerId: "flying",
    name: "Blair",
    connected: true,
    seat: 1,
    isHost: false,
    canReconnect: true,
  },
];

const room: RoomSnapshot = {
  code: "ROOM",
  phase: "round_over",
  hostPlayerId: "winner",
  players,
};

const game: GameSnapshot = {
  phase: 2,
  round: 4,
  dealerIndex: 0,
  roundWinnerIndex: 0,
  turn: {
    number: 8,
    playerIndex: 0,
    playerId: "winner",
    hasDrawn: false,
    mustUseDiscardDraw: false,
  },
  players: [
    {
      playerId: "winner",
      handCount: 0,
      totalPoints: 90,
      pointsGained: 0,
      unadjustedTotalPoints: 90,
      hasOpened: true,
    },
    {
      playerId: "flying",
      handCount: 2,
      hand: [],
      totalPoints: 89,
      pointsGained: 12,
      unadjustedTotalPoints: 107,
      hasOpened: true,
    },
  ],
  hand: [],
  drawPileCount: 0,
  discardPile: [],
  activeCompositions: [],
};

describe("GameResultsView score reveal", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    fireCelebrationConfettiMock.mockClear();
    fireStreamingCelebrationConfettiMock.mockClear();
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("renders previous, over-100, and adjusted score states in order", async () => {
    const view = render(
      <GameResultsView
        room={room}
        game={game}
        players={players}
        playerId="flying"
        connectedPlayers={2}
        dealChoice={{
          pendingDealChoice: null,
          dealChooserName: null,
          isDealChooser: false,
        }}
        onChooseDealing={() => {}}
        onSendEmote={() => {}}
      />,
    );

    const flyingScore = view.container.querySelector<HTMLElement>('[data-flying="true"]');
    expect(flyingScore?.dataset.scorePhase).toBe("previous");
    expect(flyingScore?.textContent).toContain("95");

    await act(async () => vi.advanceTimersByTime(700));
    expect(flyingScore?.dataset.scorePhase).toBe("counting");
    expect(flyingScore?.querySelector("[data-score-total]")?.textContent).toBe("95");
    expect(
      flyingScore
        ?.querySelector("[data-score-total] [data-number-trend]")
        ?.getAttribute("data-number-trend"),
    ).toBe("1");
    expect(view.container.querySelector("[data-points-arrival]")).toBeNull();

    await act(async () => vi.advanceTimersByTime(310));
    const halfwayScore = Number(
      flyingScore?.querySelector("[data-score-total]")?.textContent ?? "0",
    );
    expect(halfwayScore).toBeGreaterThan(95);
    expect(halfwayScore).toBeLessThan(107);

    await act(async () => vi.advanceTimersByTime(310));
    expect(flyingScore?.dataset.scorePhase).toBe("round");
    expect(flyingScore?.textContent).toContain("107");

    await act(async () => vi.advanceTimersByTime(180));
    expect(flyingScore?.dataset.scorePhase).toBe("adjusting");
    expect(
      flyingScore
        ?.querySelector("[data-score-total] [data-number-trend]")
        ?.getAttribute("data-number-trend"),
    ).toBe("-1");
    expect(flyingScore?.querySelector("[data-flying-icon]")).toBeNull();

    await act(async () => vi.advanceTimersByTime(260));
    const halfwayAdjustment = Number(
      flyingScore?.querySelector("[data-score-total]")?.textContent ?? "0",
    );
    expect(halfwayAdjustment).toBeGreaterThan(89);
    expect(halfwayAdjustment).toBeLessThan(107);
    expect(flyingScore?.querySelector("[data-flying-icon]")).toBeNull();

    await act(async () => vi.advanceTimersByTime(260));
    expect(flyingScore?.dataset.scorePhase).toBe("adjusted");
    expect(flyingScore?.textContent).toContain("89");
    expect(flyingScore?.textContent).not.toContain("Flew");
    expect(
      flyingScore?.querySelector('[title="Over 100: adjusted from 107 to 89 points"]'),
    ).not.toBeNull();
    expect(flyingScore?.querySelector("[data-flying-icon]")?.classList.contains("size-4")).toBe(
      true,
    );
    expect(
      flyingScore?.querySelector("[data-flying-icon]")?.classList.contains("flying-score-plane"),
    ).toBe(true);
    expect(
      flyingScore
        ?.querySelector("[data-flying-icon]")
        ?.parentElement?.classList.contains("right-[calc(100%+0.25rem)]"),
    ).toBe(true);
    expect(flyingScore?.querySelector("[data-score-total]")?.classList.contains("relative")).toBe(
      true,
    );
    expect(flyingScore?.querySelector("[data-flying-ring]")).toBeNull();
    expect(flyingScore?.querySelector("[data-flying-value]")?.classList.contains("size-9")).toBe(
      false,
    );
    expect(
      flyingScore?.querySelector("[data-score-value-slot]")?.classList.contains("w-[3ch]"),
    ).toBe(true);
    expect(
      flyingScore?.querySelector("[data-score-value-slot]")?.classList.contains("text-right"),
    ).toBe(true);
    expect(
      flyingScore?.parentElement
        ?.querySelector("[data-round-value-slot]")
        ?.classList.contains("w-[4ch]"),
    ).toBe(true);

    await act(async () => vi.advanceTimersByTime(500));
    expect(
      [...view.container.querySelectorAll('[data-slot="result-row"]')].map((row) =>
        row.getAttribute("data-player-id"),
      ),
    ).toEqual(["flying", "winner"]);
    expect(
      view.container
        .querySelector('[data-player-id="flying"]')
        ?.getAttribute("data-rankings-reordered"),
    ).toBe("true");
  });

  it("starts the score reveal after skipping the winner takeover", async () => {
    const onBackToLobby = vi.fn();
    const view = render(
      <GameResultsView
        room={{ ...room, phase: "game_over" }}
        game={game}
        players={players}
        playerId="winner"
        connectedPlayers={2}
        dealChoice={{
          pendingDealChoice: null,
          dealChooserName: null,
          isDealChooser: false,
        }}
        onBackToLobby={onBackToLobby}
        onChooseDealing={() => {}}
        onSendEmote={() => {}}
      />,
    );

    expect(document.querySelector('[data-slot="game-winner-takeover"]')).not.toBeNull();
    expect(document.querySelector('[data-slot="dialog-overlay"]')).toBeNull();
    expect(fireStreamingCelebrationConfettiMock).toHaveBeenCalledWith({
      durationMs: 3_200,
      delayMs: 260,
    });
    expect(fireCelebrationConfettiMock).not.toHaveBeenCalled();

    const flyingScore = view.container.querySelector<HTMLElement>('[data-flying="true"]');
    expect(flyingScore?.dataset.scorePhase).toBe("previous");
    await act(async () => vi.advanceTimersByTime(1_000));
    expect(flyingScore?.dataset.scorePhase).toBe("previous");

    fireEvent.click(view.getByRole("button", { name: "Final scores" }));
    await act(async () => vi.advanceTimersByTime(699));
    expect(flyingScore?.dataset.scorePhase).toBe("previous");
    await act(async () => vi.advanceTimersByTime(1));
    expect(flyingScore?.dataset.scorePhase).toBe("counting");

    fireEvent.click(view.getByRole("button", { name: "Back to lobby" }));

    expect(onBackToLobby).toHaveBeenCalledOnce();
  });

  it("starts the score reveal after the winner takeover finishes", async () => {
    const view = render(
      <GameResultsView
        room={{ ...room, phase: "game_over" }}
        game={game}
        players={players}
        playerId="winner"
        connectedPlayers={2}
        dealChoice={{
          pendingDealChoice: null,
          dealChooserName: null,
          isDealChooser: false,
        }}
        onChooseDealing={() => {}}
        onSendEmote={() => {}}
      />,
    );

    const flyingScore = view.container.querySelector<HTMLElement>('[data-flying="true"]');
    await act(async () => vi.advanceTimersByTime(4_599));
    expect(flyingScore?.dataset.scorePhase).toBe("previous");

    await act(async () => vi.advanceTimersByTime(1));
    expect(flyingScore?.dataset.scorePhase).toBe("previous");
    await act(async () => vi.advanceTimersByTime(699));
    expect(flyingScore?.dataset.scorePhase).toBe("previous");
    await act(async () => vi.advanceTimersByTime(1));
    expect(flyingScore?.dataset.scorePhase).toBe("counting");
  });

  it("does not show the lobby action after a completed round", () => {
    const view = render(
      <GameResultsView
        room={room}
        game={game}
        players={players}
        playerId="winner"
        connectedPlayers={2}
        dealChoice={{
          pendingDealChoice: null,
          dealChooserName: null,
          isDealChooser: false,
        }}
        onBackToLobby={vi.fn()}
        onChooseDealing={() => {}}
        onSendEmote={() => {}}
      />,
    );

    expect(view.queryByRole("button", { name: "Back to lobby" })).toBeNull();
    expect(view.getByRole("button", { name: "Start next round" })).not.toBeNull();
  });
});

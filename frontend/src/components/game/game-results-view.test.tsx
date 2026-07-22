// @vitest-environment jsdom

import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  type GameSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";

vi.mock("#/components/ui/animated-number", () => ({
  AnimatedNumber: ({ value, prefix }: { value: number; prefix?: string }) => (
    <span>
      {prefix}
      {value}
    </span>
  ),
}));

vi.mock("#/components/game/game-board-players", () => ({
  GameBoardPlayers: () => null,
}));

vi.mock("#/lib/confetti", () => ({
  fireCelebrationConfetti: vi.fn(),
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
      totalPoints: 40,
      pointsGained: 0,
      unadjustedTotalPoints: 40,
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

describe("GameResultsView flying score reveal", () => {
  beforeEach(() => vi.useFakeTimers());

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

    await act(async () => vi.advanceTimersByTime(500));
    expect(flyingScore?.dataset.scorePhase).toBe("round");
    expect(flyingScore?.textContent).toContain("107");

    await act(async () => vi.advanceTimersByTime(1_000));
    expect(flyingScore?.dataset.scorePhase).toBe("adjusted");
    expect(flyingScore?.textContent).toContain("89");
    expect(flyingScore?.textContent).not.toContain("Flew");
    expect(
      flyingScore?.querySelector('[title="Over 100: adjusted from 107 to 89 points"]'),
    ).not.toBeNull();
    expect(flyingScore?.querySelector("svg")).not.toBeNull();
  });

  it("returns to the lobby from final results", () => {
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

    fireEvent.click(view.getByRole("button", { name: "Back to lobby" }));

    expect(onBackToLobby).toHaveBeenCalledOnce();
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

// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";

afterEach(cleanup);

describe("GameBoardPlayers", () => {
  it("opens the current results table from the icon button", () => {
    const players: PlayerSnapshot[] = [
      {
        playerId: "player-1",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: true,
      },
      {
        playerId: "player-2",
        name: "Blair",
        connected: true,
        seat: 1,
        isHost: false,
        canReconnect: true,
      },
    ];
    const game = {
      turn: { playerId: "player-1" },
      players: [
        { playerId: "player-1", handCount: 6, totalPoints: 42 },
        { playerId: "player-2", handCount: 9, totalPoints: 18 },
      ],
    } as GameSnapshot;

    render(
      <GameBoardPlayers
        players={players}
        game={game}
        connectedPlayers={2}
        hasDraftedCompositions={false}
        onResetDraftCompositions={() => {}}
        onSendEmote={() => {}}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Results" }));

    const table = screen.getByRole("table");
    expect(within(table).getByText("Blair")).toBeTruthy();
    expect(within(table).getByText("18")).toBeTruthy();
    expect(screen.queryByText("online")).toBeNull();
  });
});

// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";

vi.mock("#/components/game-websocket-provider", () => ({
  useGameWebSocket: vi.fn(),
}));

const { ConnectionRecoveryDialog } = await import("#/components/connection-recovery-dialog");

afterEach(cleanup);

describe("ConnectionRecoveryDialog", () => {
  it("opens during recovery and closes after the connection is restored", () => {
    const view = render(<ConnectionRecoveryDialog connectionStatus="connecting" />);

    expect(screen.queryByRole("dialog")).toBeNull();

    view.rerender(<ConnectionRecoveryDialog connectionStatus="disconnected" />);
    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.queryByText("Waiting for the game server…")).toBeNull();
    expect(
      screen.getByText(
        "Still reconnecting after a few minutes? The server may be unavailable. Please try again later.",
      ),
    ).toBeTruthy();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.getByRole("dialog")).toBeTruthy();

    view.rerender(<ConnectionRecoveryDialog connectionStatus="connected" />);
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});

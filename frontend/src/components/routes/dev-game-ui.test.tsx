// @vitest-environment jsdom
import { type ComponentProps } from "react";
import { act, cleanup, render } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { type GameBoardView } from "#/components/game/game-board-view";

let board: ComponentProps<typeof GameBoardView>;
vi.mock("#/components/game/game-board-view", () => ({
  GameBoardView: (props: ComponentProps<typeof GameBoardView>) => {
    board = props;
    return null;
  },
}));
const { DevGameUi } = await import("#/components/routes/dev-game-ui");
afterEach(cleanup);

it("discards the selected card by value when its hand index is stale", async () => {
  render(<DevGameUi />);
  const hand = board!.game!.hand;
  const selected = hand.at(-1)!;
  expect(selected).not.toEqual(hand[0]);
  await act(async () => {
    await board.onDiscardCard(0, { ...selected });
  });
  expect(board!.game!.hand).toHaveLength(hand.length - 1);
  expect(board!.game!.hand[0]).toEqual(hand[0]);
  expect(board!.game!.discardPile[0]).toEqual(selected);
});

it("removes only the submitted copy of a card from the development hand", async () => {
  render(<DevGameUi />);
  const hand = board!.game!.hand;
  const selected = hand[0]!;
  await act(async () => {
    await board.onPlayTable({
      compositions: [{ cards: [{ ...selected }] }],
      additions: [],
      reclaims: [],
    });
  });
  expect(board!.game!.hand).toEqual(hand.slice(1));
  expect(board!.game!.activeCompositions.at(-1)?.cards).toEqual([selected]);
});

// @vitest-environment jsdom

import { DndContext } from "@dnd-kit/core";
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { GameBoardTable, draftPreviewForComposition } from "#/components/game/game-board-table";

afterEach(cleanup);

function renderDraftTable(showDraftTotal: boolean) {
  return render(
    <DndContext>
      <GameBoardTable
        tableCompositions={[]}
        newCompositions={[]}
        players={[]}
        turnActivity={{
          playerId: "player-1",
          round: 1,
          turnNumber: 1,
          draftCompositions: [
            {
              cards: [
                { rank: 3, suit: 0 },
                { rank: 4, suit: 0 },
                { rank: 5, suit: 0 },
              ],
            },
          ],
        }}
        canCompose={false}
        showDraftTotal={showDraftTotal}
      />
    </DndContext>,
  );
}

describe("draftPreviewForComposition", () => {
  it("shows multiple additions on the start edge in their resulting table order", () => {
    const preview = draftPreviewForComposition(
      {
        tableIndex: 0,
        key: "table-0",
        snapshot: {
          type: "run",
          cards: [
            { rank: 5, suit: 0 },
            { rank: 6, suit: 0 },
            { rank: 7, suit: 0 },
          ],
          jokerRepresentations: {},
          points: 18,
          complete: false,
        },
        stagedEntries: [],
        reclaims: [],
        insertIndex: 3,
      },
      {
        tableIndex: 0,
        insertIndex: 0,
        cardInsertIndices: { "4-0-1": 0, "3-0-1": 0 },
        cards: [
          { rank: 4, suit: 0 },
          { rank: 3, suit: 0 },
        ],
      },
      [
        { rank: 4, suit: 0 },
        { rank: 3, suit: 0 },
      ],
    );

    expect(preview.stagedEntries.map((entry) => entry.card.rank)).toEqual([3, 4]);
  });
});

describe("GameBoardTable draft total", () => {
  it("shows the total before the player has opened", () => {
    const view = renderDraftTable(true);

    expect(view.getByText(/Draft total/)).toBeTruthy();
  });

  it("hides the total after the player has opened", () => {
    const view = renderDraftTable(false);

    expect(view.queryByText(/Draft total/)).toBeNull();
  });
});

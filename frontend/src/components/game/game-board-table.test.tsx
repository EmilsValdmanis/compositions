import { describe, expect, it } from "vite-plus/test";
import { draftPreviewForComposition } from "#/components/game/game-board-table";

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

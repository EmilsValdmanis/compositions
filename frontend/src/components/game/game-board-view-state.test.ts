import { describe, expect, it } from "vitest";
import { applyHandEntryOrder, buildHandEntries } from "#/components/game/game-board-view-state";

describe("applyHandEntryOrder", () => {
  it("restores a persisted hand order and appends new cards", () => {
    const entries = buildHandEntries([
      { rank: 1, suit: 1 },
      { rank: 2, suit: 1 },
      { rank: 3, suit: 1 },
    ]);

    const reordered = applyHandEntryOrder(entries, [entries[2].key, entries[0].key]);

    expect(reordered.map((entry) => entry.key)).toEqual([
      entries[2].key,
      entries[0].key,
      entries[1].key,
    ]);
  });
});

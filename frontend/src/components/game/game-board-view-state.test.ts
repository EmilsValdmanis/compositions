import { describe, expect, it } from "vitest";
import {
  applyHandEntryOrder,
  buildHandEntries,
  buildTablePlayRequest,
  inferPlannedJokerReclaims,
  insertHandKeyIntoDraft,
} from "#/components/game/game-board-view-state";

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

describe("inferPlannedJokerReclaims", () => {
  it("treats matching cards as joker reclaims instead of additions", () => {
    const entries = buildHandEntries([
      { rank: 8, suit: 3 },
      { rank: 5, suit: 0 },
    ]);

    const result = inferPlannedJokerReclaims(
      {
        type: "set",
        cards: [
          { rank: 8, suit: 0 },
          { rank: 8, suit: 2 },
          { isJoker: true },
          { rank: 8, suit: 1 },
        ],
        jokerRepresentations: {
          2: [{ rank: 8, suit: 3 }],
        },
        points: 32,
        complete: false,
      },
      entries,
    );

    expect(result.reclaims).toHaveLength(1);
    expect(result.reclaims[0]?.jokerIndex).toBe(2);
    expect(result.reclaims[0]?.replacementEntry.key).toBe(entries[0]?.key);
    expect(result.additions.map((entry) => entry.key)).toEqual([entries[1]?.key]);
  });
});

describe("buildTablePlayRequest", () => {
  it("sends reclaims separately from additions", () => {
    const entries = buildHandEntries([
      { rank: 8, suit: 3 },
      { rank: 5, suit: 0 },
    ]);

    const request = buildTablePlayRequest(
      [
        {
          type: "set",
          cards: [
            { rank: 8, suit: 0 },
            { rank: 8, suit: 2 },
            { isJoker: true },
            { rank: 8, suit: 1 },
          ],
          jokerRepresentations: {
            2: [{ rank: 8, suit: 3 }],
          },
          points: 32,
          complete: false,
        },
      ],
      [
        {
          id: "draft-1",
          tableIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          entries,
        },
      ],
    );

    expect(request.compositions).toEqual([]);
    expect(request.reclaims).toEqual([
      {
        compositionIndex: 0,
        jokerIndex: 2,
        replacementCard: { rank: 8, suit: 3 },
      },
    ]);
    expect(request.additions).toEqual([
      {
        compositionIndex: 0,
        cards: [{ rank: 5, suit: 0 }],
      },
    ]);
  });
});

describe("insertHandKeyIntoDraft", () => {
  it("reorders cards within the same draft composition", () => {
    const entries = buildHandEntries([
      { rank: 1, suit: 1 },
      { rank: 2, suit: 1 },
      { rank: 3, suit: 1 },
    ]);

    const reordered = insertHandKeyIntoDraft(
      [
        {
          id: "draft-1",
          tableIndex: null,
          handKeys: entries.map((entry) => entry.key),
        },
      ],
      entries[0]!.key,
      "draft-1",
      entries[2]!.key,
    );

    expect(reordered[0]?.handKeys).toEqual([entries[1]!.key, entries[2]!.key, entries[0]!.key]);
  });
});

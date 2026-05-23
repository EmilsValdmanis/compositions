import { describe, expect, it } from "vitest";
import {
  applyHandEntryOrder,
  buildHandEntries,
  buildTablePlayRequest,
  buildTableCompositionViews,
  buildVirtualReclaimedJokers,
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

  it("treats matching run cards as reclaims at the end of the run", () => {
    const entries = buildHandEntries([{ rank: 6, suit: 0 }]);

    const result = inferPlannedJokerReclaims(
      {
        type: "run",
        cards: [{ rank: 5, suit: 0 }, { isJoker: true }, { rank: 7, suit: 0 }],
        jokerRepresentations: {
          1: [{ rank: 6, suit: 0 }],
        },
        points: 18,
        complete: false,
      },
      entries,
      3,
    );

    expect(result.reclaims).toHaveLength(1);
    expect(result.reclaims[0]?.jokerIndex).toBe(1);
    expect(result.reclaims[0]?.replacementEntry.key).toBe(entries[0]?.key);
  });

  it("does not allow run reclaims from the middle", () => {
    const entries = buildHandEntries([{ rank: 6, suit: 0 }]);

    const result = inferPlannedJokerReclaims(
      {
        type: "run",
        cards: [{ rank: 5, suit: 0 }, { isJoker: true }, { rank: 7, suit: 0 }],
        jokerRepresentations: {
          1: [{ rank: 6, suit: 0 }],
        },
        points: 18,
        complete: false,
      },
      entries,
      1,
    );

    expect(result.reclaims).toEqual([]);
    expect(result.additions.map((entry) => entry.key)).toEqual([entries[0]!.key]);
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
        insertIndex: 4,
        cards: [{ rank: 5, suit: 0 }],
      },
    ]);
  });

  it("preserves insert position and reuses reclaimed joker for duplicate-card cases", () => {
    const entries = buildHandEntries([
      { rank: 1, suit: 0 },
      { rank: 1, suit: 3 },
    ]);

    const request = buildTablePlayRequest(
      [
        {
          type: "run",
          cards: [{ rank: 1, suit: 0 }, { rank: 1, suit: 1 }, { isJoker: true }],
          jokerRepresentations: {
            2: [{ rank: 1, suit: 3 }],
          },
          points: 30,
          complete: false,
        },
      ],
      [
        {
          id: "draft-1",
          tableIndex: 0,
          insertIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          entries,
        },
      ],
    );

    expect(request.reclaims).toEqual([
      {
        compositionIndex: 0,
        jokerIndex: 2,
        replacementCard: { rank: 1, suit: 3 },
      },
    ]);
    expect(request.additions).toEqual([
      {
        compositionIndex: 0,
        insertIndex: 0,
        cards: [{ rank: 1, suit: 0 }],
      },
    ]);
  });

  it("allows a staged reclaimed joker to be used in a new composition", () => {
    const entries = buildHandEntries([
      { rank: 10, suit: 2 },
      { rank: 11, suit: 2 },
      { rank: 12, suit: 2 },
    ]);

    const request = buildTablePlayRequest(
      [
        {
          type: "run",
          cards: [{ rank: 8, suit: 0 }, { isJoker: true }, { rank: 10, suit: 0 }],
          jokerRepresentations: {
            1: [{ rank: 9, suit: 0 }],
          },
          points: 27,
          complete: false,
        },
      ],
      [
        {
          id: "draft-reclaim",
          tableIndex: 0,
          handKeys: ["9-0-1"],
          entries: [{ key: "9-0-1", card: { rank: 9, suit: 0 }, sourceIndex: 0 }],
        },
        {
          id: "draft-new",
          tableIndex: null,
          handKeys: [...entries.map((entry) => entry.key), "reclaimed-joker-0-1"],
          entries: [
            ...entries,
            {
              key: "reclaimed-joker-0-1",
              card: { isJoker: true },
              sourceIndex: -1,
              isVirtual: true,
            },
          ],
        },
      ],
    );

    expect(request.reclaims).toEqual([
      {
        compositionIndex: 0,
        jokerIndex: 1,
        replacementCard: { rank: 9, suit: 0 },
      },
    ]);
    expect(request.compositions).toEqual([
      {
        cards: [
          { rank: 10, suit: 2 },
          { rank: 11, suit: 2 },
          { rank: 12, suit: 2 },
          { isJoker: true },
        ],
      },
    ]);
  });

  it("reverses repeated prepend additions so runs stay in sequence", () => {
    const entries = buildHandEntries([
      { rank: 4, suit: 0 },
      { rank: 3, suit: 0 },
      { rank: 2, suit: 0 },
    ]);

    const request = buildTablePlayRequest(
      [
        {
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
      ],
      [
        {
          id: "draft-1",
          tableIndex: 0,
          insertIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          entries,
          cardInsertIndices: Object.fromEntries(entries.map((entry) => [entry.key, 0])),
        },
      ],
    );

    expect(request.additions).toEqual([
      {
        compositionIndex: 0,
        insertIndex: 0,
        cards: [
          { rank: 2, suit: 0 },
          { rank: 3, suit: 0 },
          { rank: 4, suit: 0 },
        ],
      },
    ]);
  });
});

describe("buildVirtualReclaimedJokers", () => {
  it("surfaces staged reclaims as temporary joker hand entries", () => {
    const entries = buildHandEntries([{ rank: 8, suit: 3 }]);
    const entryByKey = new Map(entries.map((entry) => [entry.key, entry]));
    const tableCompositions = buildTableCompositionViews(
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
          handKeys: [entries[0]!.key],
        },
      ],
      entryByKey,
    );

    const virtualJokers = buildVirtualReclaimedJokers(tableCompositions);

    expect(virtualJokers).toHaveLength(1);
    expect(virtualJokers[0]?.entry.card).toEqual({ isJoker: true });
    expect(virtualJokers[0]?.entry.isVirtual).toBe(true);
    expect(virtualJokers[0]?.entry.sourceIndex).toBe(-1);
    expect(virtualJokers[0]?.entry.key).toBe("reclaimed-joker-0-2");
  });

  it("orders repeated prepend additions from lowest to highest in the preview", () => {
    const entries = buildHandEntries([
      { rank: 4, suit: 0 },
      { rank: 3, suit: 0 },
      { rank: 2, suit: 0 },
    ]);
    const entryByKey = new Map(entries.map((entry) => [entry.key, entry]));
    const tableCompositions = buildTableCompositionViews(
      [
        {
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
      ],
      [
        {
          id: "draft-1",
          tableIndex: 0,
          insertIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          cardInsertIndices: Object.fromEntries(entries.map((entry) => [entry.key, 0])),
        },
      ],
      entryByKey,
    );

    expect(tableCompositions[0]?.stagedEntries.map((entry) => entry.card.rank)).toEqual([2, 3, 4]);
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

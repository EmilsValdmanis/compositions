import { describe, expect, it } from "vite-plus/test";
import {
  applyHandEntryOrder,
  buildHandEntries,
  buildTablePlayRequest,
  buildTableCompositionViews,
  buildVirtualReclaimedJokers,
  handIndexAfterSubmittedDrafts,
  inferPlannedJokerReclaims,
  insertHandKeyIntoDraft,
  moveHandEntry,
  removeHandKeyFromDrafts,
  resolveDraftViews,
  validateDraftedCompositions,
  validateOpeningTablePlay,
} from "#/components/game/game-board-view-state";

describe("validateDraftedCompositions", () => {
  it("identifies every invalid new composition", () => {
    const firstEntries = buildHandEntries([
      { rank: 5, suit: 0 },
      { rank: 7, suit: 0 },
      { rank: 8, suit: 0 },
    ]);
    const secondEntries = buildHandEntries([
      { rank: 3, suit: 1 },
      { rank: 6, suit: 1 },
      { rank: 9, suit: 1 },
    ]);

    const result = validateDraftedCompositions(
      [],
      [
        {
          id: "first",
          tableIndex: null,
          handKeys: firstEntries.map((entry) => entry.key),
          entries: firstEntries,
        },
        {
          id: "second",
          tableIndex: null,
          handKeys: secondEntries.map((entry) => entry.key),
          entries: secondEntries,
        },
      ],
    );

    expect([...result.invalidCompositionIds]).toEqual(["first", "second"]);
  });

  it("marks only the addition that makes an existing composition invalid", () => {
    const entries = buildHandEntries([
      { rank: 8, suit: 0 },
      { rank: 10, suit: 0 },
    ]);
    const result = validateDraftedCompositions(
      [
        {
          type: "run",
          cards: [
            { rank: 5, suit: 0 },
            { rank: 6, suit: 0 },
            { rank: 7, suit: 0 },
          ],
          points: 18,
          complete: false,
        },
      ],
      [
        {
          id: "addition",
          tableIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          entries,
        },
      ],
    );

    expect([...result.invalidEntryKeys]).toEqual([entries[1]!.key]);
  });
});

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

describe("handIndexAfterSubmittedDrafts", () => {
  it("uses raw hand order after staged cards are removed from a reordered hand", () => {
    const rawEntries = buildHandEntries([
      { rank: 1, suit: 0 },
      { rank: 13, suit: 3 },
      { rank: 12, suit: 2 },
      { rank: 11, suit: 1 },
    ]);
    const visuallyReorderedEntries = applyHandEntryOrder(rawEntries, [
      rawEntries[2]!.key,
      rawEntries[0]!.key,
      rawEntries[1]!.key,
      rawEntries[3]!.key,
    ]);

    const discardIndex = handIndexAfterSubmittedDrafts(rawEntries, rawEntries[2]!.key, [
      {
        id: "draft-1",
        tableIndex: null,
        handKeys: [rawEntries[0]!.key],
        entries: [rawEntries[0]!],
      },
    ]);

    expect(visuallyReorderedEntries[0]?.key).toBe(rawEntries[2]?.key);
    expect(discardIndex).toBe(1);
  });

  it("does not return an index for a card already staged for play", () => {
    const rawEntries = buildHandEntries([
      { rank: 1, suit: 0 },
      { rank: 13, suit: 3 },
    ]);

    const discardIndex = handIndexAfterSubmittedDrafts(rawEntries, rawEntries[0]!.key, [
      {
        id: "draft-1",
        tableIndex: null,
        handKeys: [rawEntries[0]!.key],
        entries: [rawEntries[0]!],
      },
    ]);

    expect(discardIndex).toBeNull();
  });

  it("accounts for every combination of staged cards around the selected card", () => {
    const rawEntries = buildHandEntries([
      { rank: 1, suit: 0 },
      { rank: 2, suit: 0 },
      { rank: 3, suit: 0 },
      { rank: 4, suit: 0 },
      { rank: 5, suit: 0 },
    ]);
    const selected = rawEntries[2]!;

    for (let stagedMask = 0; stagedMask < 1 << rawEntries.length; stagedMask += 1) {
      if ((stagedMask & (1 << selected.sourceIndex)) !== 0) {
        continue;
      }

      const stagedEntries = rawEntries.filter(
        (entry) => (stagedMask & (1 << entry.sourceIndex)) !== 0,
      );
      const expectedIndex = rawEntries
        .filter((entry) => !stagedEntries.includes(entry))
        .findIndex((entry) => entry.key === selected.key);
      const actualIndex = handIndexAfterSubmittedDrafts(rawEntries, selected.key, [
        {
          id: "draft-1",
          tableIndex: null,
          handKeys: stagedEntries.map((entry) => entry.key),
          entries: stagedEntries,
        },
      ]);

      expect(actualIndex).toBe(expectedIndex);
    }
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
          reclaimTargets: { [entries[0]!.key]: 2 },
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
          reclaimTargets: { [entries[1]!.key]: 2 },
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

  it("adds one ace before reclaiming an ambiguous set joker with the other ace", () => {
    const entries = buildHandEntries([
      { rank: 1, suit: 2 },
      { rank: 1, suit: 3 },
    ]);

    const request = buildTablePlayRequest(
      [
        {
          type: "set",
          cards: [{ rank: 1, suit: 0 }, { rank: 1, suit: 1 }, { isJoker: true }],
          jokerRepresentations: {
            2: [
              { rank: 1, suit: 2 },
              { rank: 1, suit: 3 },
            ],
          },
          points: 30,
          complete: false,
        },
      ],
      [
        {
          id: "draft-1",
          tableIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          entries,
          reclaimTargets: { [entries[1]!.key]: 2 },
        },
      ],
    );

    expect(request.additions).toEqual([
      {
        compositionIndex: 0,
        insertIndex: 3,
        cards: [{ rank: 1, suit: 2 }],
      },
    ]);
    expect(request.reclaims).toEqual([
      {
        compositionIndex: 0,
        jokerIndex: 2,
        replacementCard: { rank: 1, suit: 3 },
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
          reclaimTargets: { "9-0-1": 1 },
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

describe("validateOpeningTablePlay", () => {
  it("does not count a composition built from a reclaimed joker as the opening composition", () => {
    const entries = buildHandEntries([
      { rank: 8, suit: 2 },
      { rank: 9, suit: 2 },
    ]);

    const validation = validateOpeningTablePlay(false, [
      {
        id: "draft-reclaim",
        tableIndex: 0,
        handKeys: ["6-0-1"],
        reclaimTargets: { "6-0-1": 1 },
        entries: [{ key: "6-0-1", card: { rank: 6, suit: 0 }, sourceIndex: 0 }],
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
    ]);

    expect(validation).toEqual({ canSubmit: false, reason: "missing-own-composition" });
  });

  it("allows an unopened player to reclaim and reuse a joker after staging an own composition", () => {
    const openingEntries = buildHandEntries([
      { rank: 10, suit: 2 },
      { rank: 11, suit: 2 },
      { rank: 12, suit: 2 },
      { isJoker: true },
    ]);
    const reclaimedJokerEntries = buildHandEntries([
      { rank: 8, suit: 1 },
      { rank: 9, suit: 1 },
    ]);

    const validation = validateOpeningTablePlay(false, [
      {
        id: "draft-opening",
        tableIndex: null,
        handKeys: openingEntries.map((entry) => entry.key),
        entries: openingEntries,
      },
      {
        id: "draft-reclaim",
        tableIndex: 0,
        handKeys: ["6-0-1"],
        reclaimTargets: { "6-0-1": 1 },
        entries: [{ key: "6-0-1", card: { rank: 6, suit: 0 }, sourceIndex: 0 }],
      },
      {
        id: "draft-reused-joker",
        tableIndex: null,
        handKeys: [...reclaimedJokerEntries.map((entry) => entry.key), "reclaimed-joker-0-1"],
        entries: [
          ...reclaimedJokerEntries,
          {
            key: "reclaimed-joker-0-1",
            card: { isJoker: true },
            sourceIndex: -1,
            isVirtual: true,
          },
        ],
      },
    ]);

    expect(validation).toEqual({ canSubmit: true });
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
          reclaimTargets: { [entries[0]!.key]: 2 },
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

  it("keeps explicit reclaim entries out of staged additions in the preview", () => {
    const entries = buildHandEntries([
      { rank: 8, suit: 3 },
      { rank: 5, suit: 0 },
    ]);
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
          handKeys: entries.map((entry) => entry.key),
          reclaimTargets: { [entries[0]!.key]: 2 },
        },
      ],
      entryByKey,
    );

    expect(tableCompositions[0]?.reclaims).toHaveLength(1);
    expect(tableCompositions[0]?.reclaims[0]?.replacementEntry.key).toBe(entries[0]?.key);
    expect(tableCompositions[0]?.stagedEntries.map((entry) => entry.key)).toEqual([
      entries[1]?.key,
      entries[0]?.key,
    ]);
  });

  it("keeps a reclaimed joker visible when it is staged into a new composition", () => {
    const handEntries = buildHandEntries([
      { rank: 10, suit: 2 },
      { rank: 11, suit: 2 },
      { rank: 12, suit: 2 },
    ]);

    const resolved = resolveDraftViews(
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
          reclaimTargets: { "9-0-1": 1 },
        },
        {
          id: "draft-new",
          tableIndex: null,
          handKeys: [...handEntries.map((entry) => entry.key), "reclaimed-joker-0-1"],
        },
      ],
      [{ key: "9-0-1", card: { rank: 9, suit: 0 }, sourceIndex: 0 }, ...handEntries],
    );

    expect(resolved.virtualReclaimedJokers).toHaveLength(1);
    expect(resolved.draftCompositions[1]?.handKeys).toContain("reclaimed-joker-0-1");
    expect(resolved.allHandEntries.some((entry) => entry.key === "reclaimed-joker-0-1")).toBe(true);
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

  it("supports returning a staged reclaim to hand before reordering", () => {
    const entries = buildHandEntries([
      { rank: 4, suit: 0 },
      { rank: 5, suit: 0 },
      { rank: 8, suit: 3 },
    ]);
    const availableHandEntries = [entries[0]!, entries[1]!];
    const draggedEntry = entries[2]!;
    const draftCompositions = [
      {
        id: "draft-1",
        tableIndex: 0,
        handKeys: [draggedEntry.key],
        reclaimTargets: { [draggedEntry.key]: 2 },
      },
    ];

    const returnedToHand = removeHandKeyFromDrafts(draftCompositions, draggedEntry.key);
    const reordered = moveHandEntry(
      [...availableHandEntries, draggedEntry],
      draggedEntry.key,
      availableHandEntries[0]!.key,
    );

    expect(returnedToHand).toEqual([]);
    expect(reordered.map((entry) => entry.key)).toEqual([
      draggedEntry.key,
      availableHandEntries[0]!.key,
      availableHandEntries[1]!.key,
    ]);
  });

  it("retargets a staged card within the same table composition", () => {
    const entries = buildHandEntries([
      { rank: 4, suit: 0 },
      { rank: 8, suit: 3 },
    ]);

    const retargeted = insertHandKeyIntoDraft(
      [
        {
          id: "draft-1",
          tableIndex: 0,
          handKeys: entries.map((entry) => entry.key),
          cardInsertIndices: { [entries[0]!.key]: 0 },
          reclaimTargets: { [entries[1]!.key]: 2 },
        },
      ],
      entries[1]!.key,
      "draft-1",
      entries[0]!.key,
    );

    expect(retargeted[0]?.handKeys).toEqual([entries[1]!.key, entries[0]!.key]);
  });
});

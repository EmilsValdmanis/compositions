import { describe, expect, it } from "vite-plus/test";
import {
  draftCompositionPointTotal,
  draftCompositionPreviewPointTotal,
  isCompleteCompositionPreview,
  isCompleteDraftComposition,
  isValidDraftComposition,
} from "#/components/game/game-card-utils";

describe("draftCompositionPointTotal", () => {
  it("scores a valid set's jokers as the represented rank", () => {
    expect(
      draftCompositionPointTotal([{ rank: 10, suit: 0 }, { rank: 10, suit: 1 }, { isJoker: true }]),
    ).toBe(30);
  });

  it("scores a valid run's joker as the represented card", () => {
    expect(
      draftCompositionPointTotal([{ rank: 12, suit: 1 }, { isJoker: true }, { rank: 1, suit: 1 }]),
    ).toBe(30);
  });

  it("matches the backend by preferring a valid set over a possible run", () => {
    expect(
      draftCompositionPointTotal([{ rank: 6, suit: 0 }, { isJoker: true }, { isJoker: true }]),
    ).toBe(18);
  });

  it("scores a run when the cards cannot form a set", () => {
    expect(
      draftCompositionPointTotal([{ rank: 6, suit: 0 }, { isJoker: true }, { rank: 8, suit: 0 }]),
    ).toBe(21);
  });

  it("scores ace-low run jokers as low cards", () => {
    expect(
      draftCompositionPointTotal([{ rank: 1, suit: 2 }, { isJoker: true }, { rank: 3, suit: 2 }]),
    ).toBe(6);
  });

  it("scores one ace low and one ace high in full runs", () => {
    expect(
      draftCompositionPointTotal([
        { rank: 1, suit: 3 },
        { rank: 2, suit: 3 },
        { rank: 3, suit: 3 },
        { rank: 4, suit: 3 },
        { rank: 5, suit: 3 },
        { rank: 6, suit: 3 },
        { rank: 7, suit: 3 },
        { rank: 8, suit: 3 },
        { rank: 9, suit: 3 },
        { rank: 10, suit: 3 },
        { rank: 11, suit: 3 },
        { rank: 12, suit: 3 },
        { rank: 13, suit: 3 },
        { rank: 1, suit: 3 },
      ]),
    ).toBe(95);
  });

  it("scores a full-run joker as the high ace when needed", () => {
    expect(
      draftCompositionPointTotal([
        { rank: 1, suit: 0 },
        { rank: 2, suit: 0 },
        { rank: 3, suit: 0 },
        { rank: 4, suit: 0 },
        { rank: 5, suit: 0 },
        { rank: 6, suit: 0 },
        { rank: 7, suit: 0 },
        { rank: 8, suit: 0 },
        { rank: 9, suit: 0 },
        { rank: 10, suit: 0 },
        { rank: 11, suit: 0 },
        { rank: 12, suit: 0 },
        { rank: 13, suit: 0 },
        { isJoker: true },
      ]),
    ).toBe(95);
  });

  it.each([
    ["one natural card", [{ rank: 6, suit: 0 }], 6],
    [
      "two natural cards",
      [
        { rank: 6, suit: 0 },
        { rank: 7, suit: 0 },
      ],
      13,
    ],
    [
      "a gapped run",
      [
        { rank: 5, suit: 0 },
        { rank: 7, suit: 0 },
        { rank: 8, suit: 0 },
      ],
      20,
    ],
    [
      "a mixed-suit run",
      [
        { rank: 5, suit: 0 },
        { rank: 6, suit: 1 },
        { rank: 7, suit: 0 },
      ],
      18,
    ],
    [
      "a duplicate-suit set",
      [
        { rank: 9, suit: 0 },
        { rank: 9, suit: 0 },
        { rank: 9, suit: 1 },
      ],
      27,
    ],
  ] as const)("estimates %s from its natural cards", (_label, cards, points) => {
    expect(draftCompositionPointTotal([...cards])).toBe(points);
  });

  it("leaves an empty draft unresolved", () => {
    expect(draftCompositionPointTotal([])).toBeNull();
  });

  it("shows a staged five as five points before its composition is complete", () => {
    expect(draftCompositionPointTotal([{ rank: 5, suit: 2 }])).toBe(5);
  });

  it.each([
    ["a natural card and a joker", [{ rank: 6, suit: 0 }, { isJoker: true }]],
    ["three jokers", [{ isJoker: true }, { isJoker: true }, { isJoker: true }]],
    ["a natural card without a rank", [{ suit: 0 }, { rank: 2, suit: 0 }, { rank: 3, suit: 0 }]],
    ["a natural card without a suit", [{ rank: 1 }, { rank: 2, suit: 0 }, { rank: 3, suit: 0 }]],
    [
      "a natural card outside the deck",
      [
        { rank: 14, suit: 0 },
        { rank: 2, suit: 0 },
        { rank: 3, suit: 0 },
      ],
    ],
  ] as const)("leaves %s unresolved", (_label, cards) => {
    expect(draftCompositionPointTotal([...cards])).toBeNull();
  });

  it("honors an existing composition's type", () => {
    const setCards = [{ rank: 6, suit: 0 }, { isJoker: true }, { isJoker: true }];

    expect(draftCompositionPointTotal(setCards, "set")).toBe(18);
    expect(draftCompositionPointTotal(setCards, "run")).toBe(21);
  });

  it("includes additions in an existing composition preview", () => {
    expect(
      draftCompositionPreviewPointTotal(
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
        [{ rank: 8, suit: 0 }],
      ),
    ).toBe(26);
  });

  it("uses a reclaimed joker's replacement when previewing additions", () => {
    expect(
      draftCompositionPreviewPointTotal(
        {
          type: "run",
          cards: [{ rank: 5, suit: 2 }, { isJoker: true }, { rank: 7, suit: 2 }],
          points: 18,
          complete: false,
        },
        [{ rank: 8, suit: 2 }],
        [{ jokerIndex: 1, replacementCard: { rank: 6, suit: 2 } }],
      ),
    ).toBe(26);
  });

  it("estimates an invalid natural-card addition while it is still being placed", () => {
    expect(
      draftCompositionPreviewPointTotal(
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
        [{ rank: 9, suit: 0 }],
      ),
    ).toBe(27);
  });
});

describe("isValidDraftComposition", () => {
  it("rejects a gapped run such as five, seven, eight", () => {
    expect(
      isValidDraftComposition([
        { rank: 5, suit: 0 },
        { rank: 7, suit: 0 },
        { rank: 8, suit: 0 },
      ]),
    ).toBe(false);
  });

  it("keeps backend-valid all-joker compositions distinct from unresolved scoring", () => {
    const cards = [{ isJoker: true }, { isJoker: true }, { isJoker: true }];

    expect(isValidDraftComposition(cards)).toBe(true);
    expect(draftCompositionPointTotal(cards)).toBeNull();
  });
});

describe("completed composition previews", () => {
  it("recognizes a natural four-suit set", () => {
    expect(
      isCompleteDraftComposition([
        { rank: 9, suit: 0 },
        { rank: 9, suit: 1 },
        { rank: 9, suit: 2 },
        { rank: 9, suit: 3 },
      ]),
    ).toBe(true);
  });

  it("does not complete a four-card set that still contains a joker", () => {
    expect(
      isCompleteDraftComposition(
        [{ rank: 9, suit: 0 }, { rank: 9, suit: 1 }, { rank: 9, suit: 2 }, { isJoker: true }],
        "set",
      ),
    ).toBe(false);
  });

  it("recognizes the final card added to a table composition", () => {
    expect(
      isCompleteCompositionPreview(
        {
          type: "set",
          cards: [
            { rank: 9, suit: 0 },
            { rank: 9, suit: 1 },
            { rank: 9, suit: 2 },
          ],
          points: 27,
          complete: false,
        },
        [{ rank: 9, suit: 3 }],
      ),
    ).toBe(true);
  });
});

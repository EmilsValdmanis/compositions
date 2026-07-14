import { describe, expect, it } from "vite-plus/test";
import {
  draftCompositionPointTotal,
  draftCompositionPreviewPointTotal,
  isValidDraftComposition,
} from "#/components/game/game-card-utils";

describe("draftCompositionPointTotal", () => {
  it("scores set jokers as the represented rank", () => {
    expect(
      draftCompositionPointTotal([{ rank: 10, suit: 0 }, { rank: 10, suit: 1 }, { isJoker: true }]),
    ).toBe(30);
  });

  it("scores run jokers as the represented card", () => {
    expect(
      draftCompositionPointTotal([{ rank: 12, suit: 1 }, { isJoker: true }, { rank: 1, suit: 1 }]),
    ).toBe(30);
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

  it("leaves an all-joker draft unresolved instead of using the round-end value", () => {
    expect(draftCompositionPointTotal([{ isJoker: true }])).toBeNull();
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
});

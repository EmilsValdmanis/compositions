// @vitest-environment jsdom

import { DndContext } from "@dnd-kit/core";
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { GameBoardTable, draftPreviewForComposition } from "#/components/game/game-board-table";
import { type CardSnapshot } from "#/components/game-websocket-provider";

afterEach(cleanup);

function renderDraftTable(
  showDraftTotal: boolean,
  cards: CardSnapshot[] = [
    { rank: 3, suit: 0 },
    { rank: 4, suit: 0 },
    { rank: 5, suit: 0 },
  ],
) {
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
              cards,
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
  it("keeps a reclaimed joker with later additions on the same edge for spectators", () => {
    const view = render(
      <DndContext>
        <GameBoardTable
          tableCompositions={[
            {
              tableIndex: 0,
              key: "table-0",
              snapshot: {
                type: "run",
                cards: [{ rank: 12, suit: 0 }, { isJoker: true }, { rank: 1, suit: 0 }],
                jokerRepresentations: { 1: [{ rank: 13, suit: 0 }] },
                points: 30,
                complete: false,
              },
              stagedEntries: [],
              reclaims: [],
              insertIndex: 3,
            },
          ]}
          newCompositions={[]}
          players={[]}
          turnActivity={{
            playerId: "player-1",
            round: 1,
            turnNumber: 1,
            draftCompositions: [
              {
                tableIndex: 0,
                insertIndex: 3,
                cardPlacements: [{ reclaimJokerIndex: 1 }, { insertIndex: 0 }, { insertIndex: 0 }],
                cards: [{ rank: 13, suit: 0 }, { isJoker: true }, { rank: 10, suit: 0 }],
              },
            ],
          }}
          canCompose={false}
          showDraftTotal={false}
        />
      </DndContext>,
    );

    expect(
      [...view.container.querySelectorAll<HTMLElement>("[aria-label]")].map((card) =>
        card.getAttribute("aria-label"),
      ),
    ).toEqual(["10 of Hearts", "Joker", "Q of Hearts", "K of Hearts", "A of Hearts"]);
  });

  it("keeps two identical jokers on their independently assigned edges", () => {
    const preview = draftPreviewForComposition(
      {
        tableIndex: 0,
        key: "table-0",
        snapshot: {
          type: "run",
          cards: [{ rank: 12, suit: 0 }, { isJoker: true }, { rank: 1, suit: 0 }],
          jokerRepresentations: { 1: [{ rank: 13, suit: 0 }] },
          points: 30,
          complete: false,
        },
        stagedEntries: [],
        reclaims: [],
        insertIndex: 3,
      },
      {
        tableIndex: 0,
        insertIndex: 3,
        cardPlacements: [
          { reclaimJokerIndex: 1 },
          { insertIndex: 3 },
          { insertIndex: 0 },
          { insertIndex: 0 },
        ],
        cards: [{ rank: 13, suit: 0 }, { isJoker: true }, { isJoker: true }, { rank: 10, suit: 0 }],
      },
      [{ rank: 13, suit: 0 }, { isJoker: true }, { isJoker: true }, { rank: 10, suit: 0 }],
    );

    expect(
      preview.stagedEntries.map((entry) => ({
        card: entry.card,
        insertIndex: preview.cardInsertIndices?.[entry.key],
      })),
    ).toEqual([
      { card: { rank: 10, suit: 0 }, insertIndex: 0 },
      { card: { isJoker: true }, insertIndex: 0 },
      { card: { isJoker: true }, insertIndex: 3 },
    ]);
    expect(preview.reclaims[0]?.replacementEntry.card).toEqual({ rank: 13, suit: 0 });
  });

  it("keeps additions on independently assigned edges", () => {
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
        insertIndex: 3,
        cardPlacements: [{ insertIndex: 0 }, { insertIndex: 3 }],
        cards: [
          { rank: 4, suit: 0 },
          { rank: 8, suit: 0 },
        ],
      },
      [
        { rank: 4, suit: 0 },
        { rank: 8, suit: 0 },
      ],
    );

    expect(preview.stagedEntries.map((entry) => preview.cardInsertIndices?.[entry.key])).toEqual([
      0, 3,
    ]);
  });

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
        cardPlacements: [{ insertIndex: 0 }, { insertIndex: 0 }],
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

  it("shows an ambiguous set draft as one addition and one joker reclaim", () => {
    const preview = draftPreviewForComposition(
      {
        tableIndex: 0,
        key: "table-0",
        snapshot: {
          type: "set",
          cards: [{ rank: 13, suit: 0 }, { rank: 13, suit: 1 }, { isJoker: true }],
          jokerRepresentations: {
            2: [
              { rank: 13, suit: 2 },
              { rank: 13, suit: 3 },
            ],
          },
          points: 30,
          complete: false,
        },
        stagedEntries: [],
        reclaims: [],
        insertIndex: 3,
      },
      {
        tableIndex: 0,
        cardPlacements: [{}, { reclaimJokerIndex: 2 }],
        cards: [
          { rank: 13, suit: 2 },
          { rank: 13, suit: 3 },
        ],
      },
      [
        { rank: 13, suit: 2 },
        { rank: 13, suit: 3 },
      ],
    );

    expect(preview.stagedEntries.map((entry) => entry.card.suit)).toEqual([2]);
    expect(preview.reclaims).toHaveLength(1);
    expect(preview.reclaims[0]?.replacementEntry.card.suit).toBe(3);
  });
});

describe("GameBoardTable draft total", () => {
  it("shows the total before the player has opened", () => {
    const view = renderDraftTable(true);

    expect(
      [...view.container.querySelectorAll('[data-slot="badge"]')].some((badge) =>
        badge.textContent?.includes("Total"),
      ),
    ).toBe(true);
  });

  it("hides the total after the player has opened", () => {
    const view = renderDraftTable(false);

    expect(
      [...view.container.querySelectorAll('[data-slot="badge"]')].some((badge) =>
        badge.textContent?.includes("Total"),
      ),
    ).toBe(false);
  });

  it("renders question-mark points for an unresolved natural-card and joker draft", () => {
    const view = renderDraftTable(false, [{ rank: 6, suit: 0 }, { isJoker: true }]);
    const unresolvedScore = view.getByTitle(
      "Complete a valid composition to resolve its point value",
    );

    expect(unresolvedScore.textContent).toBe("?");
    expect(unresolvedScore.parentElement?.textContent).toContain("? pts");
  });

  it("renders a natural card's face value before the composition is valid", () => {
    const view = renderDraftTable(false, [{ rank: 5, suit: 0 }]);

    expect(
      [...view.container.querySelectorAll('[data-slot="badge"]')].some(
        (badge) => badge.textContent?.includes("5") && badge.textContent?.includes("pts"),
      ),
    ).toBe(true);
    expect(view.queryByTitle("Complete a valid composition to resolve its point value")).toBeNull();
  });
});

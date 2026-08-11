// @vitest-environment jsdom

import { DndContext } from "@dnd-kit/core";
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import { GameBoardTable } from "#/components/game/game-board-table";
import { draftPreviewForComposition } from "#/components/game/game-board-table-state";
import { type CardSnapshot } from "#/components/game-websocket-provider";

afterEach(cleanup);

function renderDraftTable(
  showDraftTotal: boolean,
  cards: CardSnapshot[] = [
    { rank: 3, suit: 0 },
    { rank: 4, suit: 0 },
    { rank: 5, suit: 0 },
  ],
  viewerPlayerId?: string,
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
              id: "draft-new-1",
              cards,
            },
          ],
        }}
        canCompose={false}
        viewerPlayerId={viewerPlayerId}
        showDraftTotal={showDraftTotal}
      />
    </DndContext>,
  );
}

describe("GameBoardTable empty state", () => {
  it("uses the Empty component when the table has no compositions", () => {
    const view = render(
      <DndContext>
        <GameBoardTable
          tableCompositions={[]}
          newCompositions={[]}
          players={[]}
          canCompose={false}
          showDraftTotal={false}
        />
      </DndContext>,
    );

    expect(view.container.querySelector('[data-slot="empty"]')).toBeTruthy();
    expect(view.getByText("No compositions on the table.")).toBeTruthy();
  });
});

describe("draftPreviewForComposition", () => {
  it("does not reserve flex gaps for empty composition edge drafts", () => {
    const view = render(
      <DndContext>
        <GameBoardTable
          tableCompositions={[
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
          ]}
          newCompositions={[]}
          players={[]}
          canCompose={false}
          showDraftTotal={false}
        />
      </DndContext>,
    );

    const edgeDraftZones = view.container.querySelectorAll(
      '[data-slot="composition-edge-draft-zone"]',
    );

    expect(edgeDraftZones).toHaveLength(2);
    expect([...edgeDraftZones].every((zone) => zone.classList.contains("contents"))).toBe(true);
  });

  it("animates additions and joker reclaims for spectators, not the acting player", () => {
    const renderPreview = (viewerPlayerId: string) =>
      render(
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
                  id: "draft-table-0",
                  tableIndex: 0,
                  insertIndex: 3,
                  cardPlacements: [
                    { reclaimJokerIndex: 1 },
                    { insertIndex: 0 },
                    { insertIndex: 0 },
                  ],
                  cards: [{ rank: 13, suit: 0 }, { isJoker: true }, { rank: 10, suit: 0 }],
                },
              ],
            }}
            canCompose={false}
            viewerPlayerId={viewerPlayerId}
            showDraftTotal={false}
          />
        </DndContext>,
      );
    const view = renderPreview("player-2");

    expect(
      [...view.container.querySelectorAll<HTMLElement>("[aria-label]")].map((card) =>
        card.getAttribute("aria-label"),
      ),
    ).toEqual(["10 of Hearts", "Joker", "Q of Hearts", "K of Hearts", "A of Hearts"]);
    expect(view.queryByText("#1")).toBeNull();
    expect(view.queryByText("30 pts")).toBeNull();
    expect(view.container.querySelectorAll('[data-spectator-card-motion="addition"]')).toHaveLength(
      2,
    );
    expect(
      view.container.querySelectorAll('[data-spectator-card-motion="joker-reclaim"]'),
    ).toHaveLength(1);

    view.unmount();
    const actorView = renderPreview("player-1");

    expect(actorView.container.querySelector("[data-spectator-card-motion]")).toBeNull();
  });

  it("includes the final added card in a completed composition's overlap", () => {
    const view = render(
      <DndContext>
        <GameBoardTable
          tableCompositions={[
            {
              tableIndex: 0,
              key: "table-0",
              snapshot: {
                type: "set",
                cards: [
                  { rank: 5, suit: 0 },
                  { rank: 5, suit: 2 },
                  { rank: 5, suit: 3 },
                ],
                points: 15,
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
                id: "draft-table-0",
                tableIndex: 0,
                cards: [{ rank: 5, suit: 1 }],
              },
            ],
          }}
          canCompose={false}
          viewerPlayerId="player-2"
          showDraftTotal={false}
        />
      </DndContext>,
    );

    const completedComposition = view.container.querySelector(
      '[data-completed-composition="true"]',
    );
    const cardWrappers = completedComposition?.querySelectorAll("[data-composition-card-wrap]");

    expect(cardWrappers).toHaveLength(4);
    expect(
      [...(cardWrappers ?? [])].map((wrapper) =>
        wrapper.getAttribute("data-composition-card-overlap"),
      ),
    ).toEqual([null, "true", "true", "true"]);
    expect(cardWrappers?.[3]?.getAttribute("data-card-suit")).toBe("1");
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
        id: "draft-table-0",
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
        id: "draft-table-0",
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
        id: "draft-table-0",
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
        id: "draft-table-0",
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
  it("marks remote draft compositions and cards for spectator motion", () => {
    const view = renderDraftTable(false, undefined, "player-2");

    expect(
      view.container.querySelectorAll('[data-spectator-composition-motion="draft"]'),
    ).toHaveLength(1);
    expect(view.container.querySelectorAll('[data-spectator-card-motion="draft"]')).toHaveLength(3);
  });

  it("does not animate draft composition placement for the acting player", () => {
    const view = renderDraftTable(false, undefined, "player-1");

    expect(view.container.querySelector('[data-spectator-composition-motion="draft"]')).toBeNull();
    expect(view.container.querySelector('[data-spectator-card-motion="draft"]')).toBeNull();
  });

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

describe("GameBoardTable mobile draft placement", () => {
  it("scrolls a newly created composition into view", () => {
    const originalMatchMedia = window.matchMedia;
    const originalScrollIntoView = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      "scrollIntoView",
    );
    const scrollIntoView = vi.fn();

    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: (query: string) => ({
        matches: query === "(max-width: 79.999rem)",
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }),
    });
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
      configurable: true,
      value: scrollIntoView,
    });

    const renderTable = (
      newCompositions: Parameters<typeof GameBoardTable>[0]["newCompositions"],
    ) => (
      <DndContext>
        <GameBoardTable
          tableCompositions={[]}
          newCompositions={newCompositions}
          players={[]}
          canCompose
          showDraftTotal={false}
        />
      </DndContext>
    );
    const view = render(renderTable([]));

    view.rerender(
      renderTable([
        {
          id: "draft-1",
          handKeys: ["3-0-1"],
          tableIndex: null,
          entries: [{ key: "3-0-1", card: { rank: 3, suit: 0 }, sourceIndex: 0 }],
        },
      ]),
    );

    expect(scrollIntoView).toHaveBeenCalledWith({
      behavior: "smooth",
      block: "nearest",
      inline: "nearest",
    });

    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: originalMatchMedia,
    });
    if (originalScrollIntoView) {
      Object.defineProperty(HTMLElement.prototype, "scrollIntoView", originalScrollIntoView);
    } else {
      Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
    }
  });
});

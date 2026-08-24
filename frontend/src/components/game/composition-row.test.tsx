// @vitest-environment jsdom

import { DndContext } from "@dnd-kit/core";
import { cleanup, render } from "@testing-library/react";
import { type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";

vi.mock("motion/react", () => ({
  AnimatePresence: ({ children }: { children: ReactNode }) => children,
  motion: {
    div: ({
      children,
      layout,
      initial: _initial,
      animate: _animate,
      exit: _exit,
      transition: _transition,
      ...props
    }: React.HTMLAttributes<HTMLDivElement> & {
      layout?: string | boolean;
      initial?: unknown;
      animate?: unknown;
      exit?: unknown;
      transition?: unknown;
    }) => (
      <div data-motion-layout={layout ? String(layout) : "false"} {...props}>
        {children}
      </div>
    ),
  },
}));

const { CompositionRow } = await import("#/components/game/composition-row");

afterEach(cleanup);

describe("CompositionRow placement motion", () => {
  it("does not layout-animate settled cards past a newly placed end card", () => {
    const view = render(
      <DndContext>
        <CompositionRow
          composition={{
            type: "run",
            cards: [
              { rank: 1, suit: 1 },
              { rank: 2, suit: 1 },
            ],
            points: 3,
            complete: false,
          }}
          index={0}
          stagedEntries={[
            {
              key: "3-1-1",
              card: { rank: 3, suit: 1 },
              sourceIndex: 0,
            },
          ]}
          cardInsertIndices={{ "3-1-1": 2 }}
          players={[]}
        />
      </DndContext>,
    );

    const settledTwo = view.container.querySelector<HTMLElement>(
      '[data-composition-card-wrap][data-card-rank="2"]',
    );

    expect(settledTwo?.getAttribute("data-motion-layout")).toBe("false");
  });

  it("leaves interactive card movement entirely to dnd-kit", () => {
    const view = render(
      <DndContext>
        <CompositionRow
          composition={{
            type: "run",
            cards: [
              { rank: 1, suit: 1 },
              { rank: 2, suit: 1 },
            ],
            points: 3,
            complete: false,
          }}
          index={0}
          stagedEntries={[
            {
              key: "3-1-1",
              card: { rank: 3, suit: 1 },
              sourceIndex: 0,
            },
          ]}
          cardInsertIndices={{ "3-1-1": 2 }}
          players={[]}
        />
      </DndContext>,
    );

    const actingPlayersCard = view.container.querySelector<HTMLElement>(
      '[data-composition-card-wrap][data-card-rank="3"]',
    );

    expect(actingPlayersCard?.getAttribute("data-motion-layout")).toBeNull();
  });

  it("keeps explanatory placement motion for viewing players", () => {
    const view = render(
      <DndContext>
        <CompositionRow
          composition={{
            type: "run",
            cards: [
              { rank: 1, suit: 1 },
              { rank: 2, suit: 1 },
            ],
            points: 3,
            complete: false,
          }}
          index={0}
          stagedEntries={[
            {
              key: "3-1-1",
              card: { rank: 3, suit: 1 },
              sourceIndex: 0,
            },
          ]}
          cardInsertIndices={{ "3-1-1": 2 }}
          players={[]}
          stagedEntriesInteractive={false}
          animateStagedEntries
        />
      </DndContext>,
    );

    const viewingPlayersCard = view.container.querySelector<HTMLElement>(
      '[data-spectator-card-motion="addition"]',
    );

    expect(viewingPlayersCard?.getAttribute("data-motion-layout")).toBe("position");
  });
});

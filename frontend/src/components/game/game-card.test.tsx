// @vitest-environment jsdom

import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";

import { GameCard } from "#/components/game/game-card";
import { setReducedMotionPreferenceEnabled } from "#/lib/reduced-motion";

afterEach(() => {
  setReducedMotionPreferenceEnabled(false);
  cleanup();
});

describe("GameCard", () => {
  it("contains face layers within the card's stacking context", () => {
    const view = render(<GameCard card={{ rank: 13, suit: 0 }} />);

    expect(view.container.querySelector("[data-game-card]")?.classList.contains("isolate")).toBe(
      true,
    );
  });

  it("does not clip activity labels positioned outside the card face", () => {
    const view = render(
      <GameCard
        card={{ rank: 11, suit: 1 }}
        decoration={{ highlight: "addition", label: <span>Add</span> }}
      />,
    );

    const card = view.container.querySelector("[data-game-card]");

    expect(view.getByText("Add")).toBeTruthy();
    expect(card?.classList.contains("overflow-hidden")).toBe(false);
  });

  it.each([
    ["hearts", 0],
    ["diamonds", 1],
    ["clubs", 2],
    ["spades", 3],
  ])("uses the same centered %s icon for number and face cards", (_name, suit) => {
    const numberCard = render(<GameCard card={{ rank: 7, suit }} />);
    const faceCard = render(<GameCard card={{ rank: 12, suit }} />);

    const numberIcon = numberCard.container.querySelector("svg");
    const faceIcon = faceCard.container.querySelector("svg");

    expect(numberIcon).not.toBeNull();
    expect(faceIcon?.innerHTML).toBe(numberIcon?.innerHTML);
    expect(faceCard.container.querySelectorAll("svg")).toHaveLength(1);
  });

  it("keeps a single centered icon on jokers", () => {
    const view = render(<GameCard card={{ isJoker: true }} />);

    expect(view.container.querySelectorAll("svg")).toHaveLength(1);
  });

  it("keeps interactive cards visible when reduced motion is enabled", () => {
    setReducedMotionPreferenceEnabled(true);

    const view = render(
      <GameCard
        card={{ rank: 7, suit: 3 }}
        dragSource={{ id: "discard-draw", data: { drawSource: "discard" } }}
      />,
    );

    const card = view.container.querySelector("button");
    expect(card?.style.opacity).toBe("1");
    expect(card?.querySelector("svg")).toBeTruthy();
  });
});

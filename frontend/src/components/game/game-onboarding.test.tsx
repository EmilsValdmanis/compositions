// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import {
  type CardSnapshot,
  type DraftCompositionSnapshot,
  type TablePlayRequest,
} from "#/components/game-websocket-provider";
import { m } from "#/paraglide/messages.js";

const { completeOnboardingMock, fireCelebrationConfettiMock } = vi.hoisted(() => ({
  completeOnboardingMock: vi.fn(),
  fireCelebrationConfettiMock: vi.fn(),
}));

vi.mock("#/lib/auth-client", () => ({
  authClient: {
    completeOnboarding: completeOnboardingMock,
  },
}));

vi.mock("#/lib/confetti", () => ({
  fireCelebrationConfetti: fireCelebrationConfettiMock,
}));

type MockBoardProps = {
  onDrawFromDeck: () => void;
  guidance?: {
    stage: "orientation" | "draw" | "compose" | "discard";
    onDrawDragStateChange: (isDragging: boolean) => void;
    onDrawSettled: () => void;
    onDraftStateChange: (state: {
      hasDraftedCompositions: boolean;
      canSubmitTablePlay: boolean;
      draftCompositions: DraftCompositionSnapshot[];
      isDraggingCard: boolean;
    }) => void;
  };
  onPlayTableAndDiscard: (
    play: TablePlayRequest,
    cardIndex: number,
    card: CardSnapshot,
  ) => Promise<unknown> | void;
};

vi.mock("#/components/game/game-board-view", () => ({
  GameBoardView: (props: MockBoardProps) => (
    <div>
      <button
        type="button"
        data-card-motion-source="deck"
        onClick={() => {
          props.onDrawFromDeck();
          props.guidance?.onDrawDragStateChange(true);
        }}
      >
        Mock start draw
      </button>
      <button
        type="button"
        onClick={() => {
          props.guidance?.onDrawDragStateChange(false);
          props.guidance?.onDrawSettled();
        }}
      >
        Mock drop draw
      </button>
      <button
        type="button"
        data-onboarding-target="table"
        style={{ borderRadius: 32 }}
        onClick={() =>
          props.guidance?.onDraftStateChange({
            hasDraftedCompositions: true,
            canSubmitTablePlay: true,
            draftCompositions: [{ id: "partial-run", cards: [{ rank: 5, suit: 3 }] }],
            isDraggingCard: false,
          })
        }
      >
        Mock partial composition
      </button>
      <button
        type="button"
        onClick={() =>
          props.guidance?.onDraftStateChange({
            hasDraftedCompositions: true,
            canSubmitTablePlay: true,
            draftCompositions: [
              {
                id: "complete-run",
                cards: [
                  { rank: 5, suit: 3 },
                  { rank: 6, suit: 3 },
                  { rank: 7, suit: 3 },
                ],
              },
            ],
            isDraggingCard: true,
          })
        }
      >
        Mock hovering final card
      </button>
      <button
        type="button"
        onClick={() =>
          props.guidance?.onDraftStateChange({
            hasDraftedCompositions: true,
            canSubmitTablePlay: true,
            draftCompositions: [
              {
                id: "complete-run",
                cards: [
                  { rank: 5, suit: 3 },
                  { rank: 6, suit: 3 },
                  { rank: 7, suit: 3 },
                ],
              },
            ],
            isDraggingCard: false,
          })
        }
      >
        Mock valid composition
      </button>
      <div data-onboarding-target="hand">
        <div data-onboarding-target="hand-cards">
          <div data-card-rank="5" data-card-suit="3" />
          <div data-card-rank="6" data-card-suit="3" />
          <div data-card-rank="12" data-card-suit="0" />
          <div data-card-rank="7" data-card-suit="3" />
        </div>
      </div>
      <div data-onboarding-target="new-composition" style={{ borderRadius: 24 }}>
        Mock composition area
      </div>
      <button
        type="button"
        data-card-motion-source="discard"
        onClick={() =>
          void props.onPlayTableAndDiscard(
            {
              compositions: [
                {
                  cards: [
                    { rank: 5, suit: 3 },
                    { rank: 6, suit: 3 },
                    { rank: 7, suit: 3 },
                  ],
                },
              ],
              additions: [],
              reclaims: [],
            },
            0,
            { rank: 12, suit: 0 },
          )
        }
      >
        Mock discard
      </button>
    </div>
  ),
}));

const { GameOnboardingProvider, useGameOnboarding } =
  await import("#/components/game/game-onboarding");

function ReplayTutorialButton() {
  const { startTutorial } = useGameOnboarding();
  return (
    <button type="button" onClick={startTutorial}>
      Replay tutorial
    </button>
  );
}

function renderTutorial(completedVersion = 0) {
  return render(
    <GameOnboardingProvider completedVersion={completedVersion} requiredVersion={1}>
      <div>Real game UI</div>
    </GameOnboardingProvider>,
  );
}

function mockRect(left: number, top: number, width: number, height: number): DOMRect {
  return {
    x: left,
    y: top,
    left,
    top,
    right: left + width,
    bottom: top + height,
    width,
    height,
    toJSON: () => ({}),
  };
}

function spotlightMaskMarkup() {
  const maskImage = document.querySelector<HTMLElement>("[data-onboarding-backdrop]")?.style
    .maskImage;
  expect(maskImage).toContain("data:image/svg+xml");
  return decodeURIComponent(maskImage ?? "");
}

function tutorialPresentation(stage: string) {
  return waitFor(() => {
    const presentation = document.querySelector<HTMLElement>(
      `[data-tutorial-presentation][data-tutorial-stage="${stage}"]`,
    );
    expect(presentation).not.toBeNull();
    return presentation!;
  });
}

beforeEach(() => {
  completeOnboardingMock.mockReset();
  completeOnboardingMock.mockResolvedValue(undefined);
  fireCelebrationConfettiMock.mockReset();
  fireCelebrationConfettiMock.mockResolvedValue(undefined);
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    if (this.matches('[data-card-motion-source="deck"]')) {
      return mockRect(100, 100, 60, 90);
    }
    if (this.matches('[data-onboarding-target="hand"]')) {
      return mockRect(100, 500, 600, 120);
    }
    if (this.matches('[data-onboarding-target="hand-cards"]')) {
      return mockRect(270, 510, 260, 100);
    }
    if (this.matches('[data-card-rank="5"][data-card-suit="3"]')) {
      return mockRect(270, 520, 56, 80);
    }
    if (this.matches('[data-card-rank="6"][data-card-suit="3"]')) {
      return mockRect(334, 520, 56, 80);
    }
    if (this.matches('[data-card-rank="7"][data-card-suit="3"]')) {
      return mockRect(462, 520, 56, 80);
    }
    if (this.matches('[data-card-motion-source="discard"]')) {
      return mockRect(180, 100, 60, 90);
    }
    if (this.matches('[data-onboarding-target="new-composition"]')) {
      return mockRect(350, 250, 160, 100);
    }
    if (this.matches('[data-onboarding-target="table"]')) {
      return mockRect(50, 200, 700, 250);
    }
    return mockRect(0, 0, 0, 0);
  });
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("GameOnboardingProvider", () => {
  it("pluralizes tutorial card progress in both languages", () => {
    expect(m.onboarding_cards_placed({ count: 1, total: 3 }, { locale: "en" })).toBe(
      "1 card placed out of 3",
    );
    expect(m.onboarding_cards_placed({ count: 2, total: 3 }, { locale: "en" })).toBe(
      "2 cards placed out of 3",
    );
    expect(m.onboarding_cards_placed({ count: 0, total: 3 }, { locale: "lv" })).toBe(
      "Nav novietota neviena no 3 kārtīm",
    );
    expect(m.onboarding_cards_placed({ count: 1, total: 3 }, { locale: "lv" })).toBe(
      "Novietota 1 kārts no 3",
    );
    expect(m.onboarding_cards_placed({ count: 2, total: 3 }, { locale: "lv" })).toBe(
      "Novietotas 2 kārtis no 3",
    );
  });

  it("opens a simplified practice coach without skip or step-count controls", async () => {
    renderTutorial();

    const dialog = await screen.findByRole("dialog", { name: "Let’s play one turn together" });
    expect(dialog).toBeTruthy();
    expect(dialog.dataset.slot).toBe("dialog-content");
    expect((await tutorialPresentation("intro")).dataset.centered).toBe("true");
    expect(dialog.className).not.toContain("overflow");
    expect(document.querySelector("[data-onboarding-backdrop]")?.className).toContain("z-20");
    expect(screen.getByLabelText("Practice game")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Skip tutorial" })).toBeNull();
    expect(screen.queryByText("1 of 5")).toBeNull();
    const stageBadge = screen.getByText("Safe practice round");
    expect(stageBadge.dataset.slot).toBe("badge");
    expect(stageBadge.parentElement?.dataset.slot).toBe("card-header");
    const coachTitle = screen
      .getAllByText("Let’s play one turn together")
      .find((element) => element.parentElement?.dataset.slot === "card-content");
    expect(coachTitle).toBeTruthy();
  });

  it("uses each account's completion version instead of shared browser storage", async () => {
    const view = render(
      <GameOnboardingProvider key="returning-player" completedVersion={1} requiredVersion={1}>
        <div>Real game UI</div>
      </GameOnboardingProvider>,
    );

    expect(screen.queryByRole("dialog")).toBeNull();

    view.rerender(
      <GameOnboardingProvider key="new-player" completedVersion={0} requiredVersion={1}>
        <div>Real game UI</div>
      </GameOnboardingProvider>,
    );

    expect(
      await screen.findByRole("dialog", { name: "Let’s play one turn together" }),
    ).toBeTruthy();
  });

  it("allows a completed account to replay the tutorial on demand", async () => {
    render(
      <GameOnboardingProvider completedVersion={1} requiredVersion={1}>
        <ReplayTutorialButton />
      </GameOnboardingProvider>,
    );

    expect(screen.queryByRole("dialog")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Replay tutorial" }));

    expect(
      await screen.findByRole("dialog", { name: "Let’s play one turn together" }),
    ).toBeTruthy();
  });

  it("keeps every action coach out of the board on compact screens", async () => {
    vi.stubGlobal("matchMedia", () => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    renderTutorial();

    await screen.findByRole("dialog", { name: "Let’s play one turn together" });
    fireEvent.click(screen.getByRole("button", { name: "Take the turn" }));

    const orientationDialog = await screen.findByRole("dialog", {
      name: "Meet your playing area",
    });
    expect(orientationDialog.getAttribute("aria-modal")).toBeNull();
    expect((await tutorialPresentation("orientation")).dataset.tutorialPresentation).toBe("inline");
    await waitFor(() =>
      expect(
        screen
          .getByText("Composition area")
          .closest<HTMLElement>("[data-onboarding-orientation-label]")?.style.top,
      ).toBe("192px"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Got it" }));
    await screen.findByRole("dialog", { name: "Pick up from the draw pile" });
    expect((await tutorialPresentation("draw")).dataset.tutorialPresentation).toBe("inline");

    fireEvent.click(screen.getByRole("button", { name: "Mock start draw" }));
    fireEvent.click(screen.getByRole("button", { name: "Mock drop draw" }));
    await screen.findByRole("dialog", { name: "Build a spade run" });
    expect((await tutorialPresentation("compose")).dataset.tutorialPresentation).toBe("inline");
    await waitFor(() =>
      expect(document.querySelectorAll("[data-onboarding-spotlight]")).toHaveLength(4),
    );
    const composeSpotlights = Array.from(
      document.querySelectorAll<HTMLElement>("[data-onboarding-spotlight]"),
    );
    expect(composeSpotlights.slice(0, 3).map((spotlight) => spotlight.style.left)).toEqual([
      "265px",
      "329px",
      "457px",
    ]);
    expect(composeSpotlights.slice(0, 3).map((spotlight) => spotlight.style.width)).toEqual([
      "66px",
      "66px",
      "66px",
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Mock valid composition" }));
    await screen.findByRole("dialog", { name: "Finish by discarding" });
    expect((await tutorialPresentation("discard")).dataset.tutorialPresentation).toBe("inline");

    fireEvent.click(screen.getByRole("button", { name: "Mock discard" }));
    expect(await screen.findByRole("dialog", { name: "You just finished a turn" })).toBeTruthy();
  });

  it("advances only after the player performs each real board action", async () => {
    renderTutorial();
    await screen.findByRole("dialog", { name: "Let’s play one turn together" });

    fireEvent.click(screen.getByRole("button", { name: "Take the turn" }));
    await screen.findByRole("dialog", {
      name: "Meet your playing area",
    });
    const orientationPresentation = await tutorialPresentation("orientation");
    expect(orientationPresentation.dataset.tutorialPresentation).toBe("overlay");
    expect(orientationPresentation.style.top).toBe("215px");
    expect(orientationPresentation.style.left).toBe("351px");
    expect(screen.getByText("Your hand")).toBeTruthy();
    expect(screen.getByText("Draw & discard")).toBeTruthy();
    expect(screen.getByText("Composition area")).toBeTruthy();
    await waitFor(() =>
      expect(document.querySelectorAll("[data-onboarding-spotlight]")).toHaveLength(3),
    );
    expect(spotlightMaskMarkup().match(/<rect [^>]*rx=/g)).toHaveLength(3);
    expect(document.querySelector<HTMLElement>("[data-onboarding-spotlight]")?.style.width).toBe(
      "270px",
    );
    expect(
      Array.from(document.querySelectorAll<HTMLElement>("[data-onboarding-spotlight]")).at(-1)
        ?.style.width,
    ).toBe("710px");
    expect(
      Array.from(document.querySelectorAll<HTMLElement>("[data-onboarding-spotlight]")).at(-1)
        ?.style.borderRadius,
    ).toBe("37px");

    fireEvent.click(screen.getByRole("button", { name: "Got it" }));
    await screen.findByRole("dialog", {
      name: "Pick up from the draw pile",
    });
    const drawPresentation = await tutorialPresentation("draw");
    await waitFor(() =>
      expect(document.querySelectorAll("[data-onboarding-spotlight]")).toHaveLength(1),
    );
    const drawCoachLeft = drawPresentation.style.left;

    fireEvent.click(screen.getByRole("button", { name: "Mock start draw" }));
    await waitFor(() =>
      expect(document.querySelectorAll("[data-onboarding-spotlight]")).toHaveLength(2),
    );
    expect((await tutorialPresentation("draw")).style.left).toBe(drawCoachLeft);
    expect(await screen.findByRole("dialog", { name: "Pick up from the draw pile" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Mock drop draw" }));
    expect((await tutorialPresentation("draw")).dataset.tutorialStage).toBe("draw");
    const composeDialog = await screen.findByRole("dialog", { name: "Build a spade run" });
    expect(composeDialog).toBeTruthy();
    expect((await tutorialPresentation("compose")).dataset.tutorialStage).toBe("compose");
    await waitFor(() =>
      expect(
        Array.from(document.querySelectorAll<HTMLElement>("[data-onboarding-spotlight]")).at(-1)
          ?.style.borderRadius,
      ).toBe("29px"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Mock partial composition" }));
    expect(await screen.findByRole("dialog", { name: "Build a spade run" })).toBeTruthy();
    expect(screen.getByText("1 card placed out of 3")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Mock hovering final card" }));
    expect(await screen.findByRole("dialog", { name: "Build a spade run" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Mock valid composition" }));
    expect((await tutorialPresentation("compose")).dataset.tutorialStage).toBe("compose");
    const discardDialog = await screen.findByRole("dialog", { name: "Finish by discarding" });
    expect(discardDialog).toBeTruthy();
    expect((await tutorialPresentation("discard")).dataset.tutorialStage).toBe("discard");

    fireEvent.click(screen.getByRole("button", { name: "Mock discard" }));
    expect((await tutorialPresentation("discard")).dataset.tutorialStage).toBe("discard");
    await screen.findByRole("dialog", {
      name: "You just finished a turn",
    });
    expect((await tutorialPresentation("complete")).dataset.tutorialStage).toBe("complete");
    expect(fireCelebrationConfettiMock).toHaveBeenCalledWith({
      count: 140,
      originY: 0.72,
      delayMs: 100,
    });

    completeOnboardingMock.mockRejectedValueOnce(new Error("offline"));
    fireEvent.click(screen.getByRole("button", { name: "Start playing" }));

    await waitFor(() => expect(completeOnboardingMock).toHaveBeenCalledOnce());
    expect(screen.getByRole("dialog", { name: "You just finished a turn" })).toBeTruthy();
    await waitFor(() =>
      expect(
        screen.getByRole<HTMLButtonElement>("button", { name: "Start playing" }).disabled,
      ).toBe(false),
    );

    completeOnboardingMock.mockResolvedValueOnce(undefined);
    fireEvent.click(screen.getByRole("button", { name: "Start playing" }));

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(completeOnboardingMock).toHaveBeenCalledTimes(2);
  });
});

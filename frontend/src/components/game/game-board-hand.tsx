import { useEffect, useRef, useState } from "react";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { ArrowLeft02Icon, ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import { Caption } from "#/components/typography";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

type HandStatus = {
  hasGame: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
};

type TablePlayState = {
  hasDraftedCompositions: boolean;
};

type GameBoardHandProps = {
  status: HandStatus;
  availableHandEntries: HandEntry[];
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  tablePlayState: TablePlayState;
};

export function GameBoardHand({
  status,
  availableHandEntries,
  sortableIds,
  activeDrag,
  tablePlayState,
}: GameBoardHandProps) {
  const { hasGame } = status;
  const { hasDraftedCompositions } = tablePlayState;
  const shouldReduceMotion = useShouldReduceMotion();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollState, setScrollState] = useState({
    hasOverflow: false,
    canScrollLeft: false,
    canScrollRight: false,
  });

  function updateScrollState() {
    const scroller = scrollRef.current;
    if (!scroller) {
      return;
    }

    const maxScrollLeft = scroller.scrollWidth - scroller.clientWidth;
    const nextState = {
      hasOverflow: maxScrollLeft > 2,
      canScrollLeft: scroller.scrollLeft > 2,
      canScrollRight: scroller.scrollLeft < maxScrollLeft - 2,
    };

    setScrollState((current) =>
      current.hasOverflow === nextState.hasOverflow &&
      current.canScrollLeft === nextState.canScrollLeft &&
      current.canScrollRight === nextState.canScrollRight
        ? current
        : nextState,
    );
  }

  useEffect(() => {
    const frame = window.requestAnimationFrame?.(updateScrollState);
    if (frame === undefined) {
      updateScrollState();
    }
    window.addEventListener("resize", updateScrollState);

    return () => {
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame);
      }
      window.removeEventListener("resize", updateScrollState);
    };
  }, [availableHandEntries.length]);

  function scrollHand(direction: -1 | 1) {
    const scroller = scrollRef.current;
    if (!scroller) {
      return;
    }

    scroller.scrollBy({
      left: direction * Math.max(scroller.clientWidth * 0.72, 160),
      behavior: shouldReduceMotion ? "auto" : "smooth",
    });
  }

  return (
    <Card className="min-h-0 shrink-0 [--card-spacing:--spacing(2)] xl:[--card-spacing:--spacing(6)]">
      <CardContent className="min-h-0">
        {hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="min-h-0 rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="min-w-0">
                  <div className="mb-1 flex min-h-6 items-center justify-between gap-2 px-1 xl:hidden">
                    <Caption className="truncate text-[0.6875rem] text-muted-foreground/80">
                      {m.hand_mobile_hint()}
                    </Caption>
                    <div
                      className={cn("flex shrink-0 gap-1", !scrollState.hasOverflow && "invisible")}
                      aria-hidden={!scrollState.hasOverflow || undefined}
                    >
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-xs"
                        disabled={!scrollState.canScrollLeft}
                        onClick={() => scrollHand(-1)}
                        aria-label={m.scroll_hand_left()}
                        title={m.scroll_hand_left()}
                      >
                        <HugeiconsIcon icon={ArrowLeft02Icon} strokeWidth={2} />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-xs"
                        disabled={!scrollState.canScrollRight}
                        onClick={() => scrollHand(1)}
                        aria-label={m.scroll_hand_right()}
                        title={m.scroll_hand_right()}
                      >
                        <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} />
                      </Button>
                    </div>
                  </div>

                  <div className="relative min-w-0">
                    <div
                      className={cn(
                        "pointer-events-none absolute inset-y-0 left-0 z-10 w-6 bg-linear-to-r from-card to-transparent transition-opacity duration-150 xl:hidden",
                        scrollState.canScrollLeft ? "opacity-100" : "opacity-0",
                      )}
                    />
                    <div
                      className={cn(
                        "pointer-events-none absolute inset-y-0 right-0 z-10 w-6 bg-linear-to-l from-card to-transparent transition-opacity duration-150 xl:hidden",
                        scrollState.canScrollRight ? "opacity-100" : "opacity-0",
                      )}
                    />
                    <div
                      ref={scrollRef}
                      onScroll={updateScrollState}
                      className="min-h-0 touch-pan-x overflow-x-auto overflow-y-hidden overscroll-x-contain px-1 pb-1"
                    >
                      <div className="flex w-max min-w-full justify-start gap-2 xl:justify-center">
                        {availableHandEntries.map((entry) => (
                          <GameCard
                            key={entry.key}
                            card={entry.card}
                            size="hand"
                            draggable={{
                              id: entry.key,
                              cardIndex: entry.sourceIndex,
                              isVirtual: entry.isVirtual,
                            }}
                            className={cn(
                              "[@media(max-height:600px)]:h-22 [@media(max-height:600px)]:w-15",
                              activeDrag?.type === "draw" &&
                                entry.key === activeDrag.revealedHandKey
                                ? "invisible"
                                : undefined,
                            )}
                          />
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <Caption className="rounded-3xl border border-dashed border-border/70 p-6">
                  {hasDraftedCompositions ? m.all_cards_staged() : m.no_cards_in_hand()}
                </Caption>
              )}
            </GameBoardDraftDropZone>
          </SortableContext>
        ) : (
          <Caption className="rounded-3xl border border-dashed border-border/70 p-6">
            {m.waiting_game_snapshot()}
          </Caption>
        )}
      </CardContent>
    </Card>
  );
}

import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { GameBoardDraftDropZone } from "#/components/game/game-board-draft-drop-zone";
import {
  type ActiveDrag,
  type HandEntry,
  HAND_DROP_ID,
} from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Card, CardContent } from "#/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia } from "#/components/ui/empty";
import { Spinner } from "#/components/ui/spinner";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

type HandStatus = {
  hasGame: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
  isSpectating?: boolean;
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

function HandState({
  children,
  loading = false,
}: {
  children: React.ReactNode;
  loading?: boolean;
}) {
  return (
    <Empty className="border p-6">
      <EmptyHeader>
        {loading ? (
          <EmptyMedia variant="icon">
            <Spinner />
          </EmptyMedia>
        ) : null}
        <EmptyDescription>{children}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

export function GameBoardHand({
  status,
  availableHandEntries,
  sortableIds,
  activeDrag,
  tablePlayState,
}: GameBoardHandProps) {
  const { hasGame } = status;
  const { hasDraftedCompositions } = tablePlayState;
  return (
    <Card
      data-onboarding-target="hand"
      className="min-h-0 shrink-0 pb-0 [--card-spacing:--spacing(2)] xl:[--card-spacing:--spacing(6)]"
    >
      <CardContent className="min-h-0 px-0">
        {status.isSpectating ? (
          <HandState>{m.spectator_hand_hidden()}</HandState>
        ) : hasGame ? (
          <SortableContext items={sortableIds} strategy={horizontalListSortingStrategy}>
            <GameBoardDraftDropZone
              id={HAND_DROP_ID}
              className="min-h-0 rounded-3xl border border-transparent"
            >
              {availableHandEntries.length ? (
                <div className="min-h-0 touch-pan-x overflow-x-auto overflow-y-hidden overscroll-x-contain px-(--card-spacing) scroll-fade-x pb-1">
                  <div className="flex min-w-full justify-center pb-(--card-spacing)">
                    <div data-onboarding-target="hand-cards" className="flex w-max shrink-0 gap-2">
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
                            activeDrag?.type === "draw" && entry.key === activeDrag.revealedHandKey
                              ? "invisible"
                              : undefined,
                          )}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              ) : (
                <HandState>
                  {hasDraftedCompositions ? m.all_cards_staged() : m.no_cards_in_hand()}
                </HandState>
              )}
            </GameBoardDraftDropZone>
          </SortableContext>
        ) : (
          <HandState loading>{m.waiting_game_snapshot()}</HandState>
        )}
      </CardContent>
    </Card>
  );
}

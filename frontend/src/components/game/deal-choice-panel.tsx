import { useEffect, useState } from "react";
import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  ArrangeIcon,
  ArrowLeft02Icon,
  ArrowRight01Icon,
  Cards02Icon,
  DragDropVerticalIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  type DealingChoiceRequest,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { cn } from "#/lib/utils";

const GAME_DECK_CARD_COUNT = 108;
const VISUAL_CARD_COUNT = 30;

type DealMode = "round_robin" | "tap";
type DealStep = "cut" | "deal";

type DealChoicePanelProps = {
  players: PlayerSnapshot[];
  pendingDealChoice: PendingDealChoiceSnapshot;
  dealChooserName: string | null;
  isDealChooser: boolean;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
};

function defaultTapOrder(players: PlayerSnapshot[], pendingDealChoice: PendingDealChoiceSnapshot) {
  return players.map((_, offset) => (pendingDealChoice.dealerIndex + offset + 1) % players.length);
}

function dealPlayerId(playerIndex: number) {
  return `deal-player-${playerIndex}`;
}

function playerIndexFromDealId(id: string) {
  return Number(id.replace("deal-player-", ""));
}

function clampCutSize(rawCutSize: string, maxCutSize: number) {
  const parsed = Number.parseInt(rawCutSize, 10);

  if (!Number.isFinite(parsed)) {
    return 0;
  }

  return Math.min(Math.max(parsed, 0), maxCutSize);
}

function DeckCard({ index, cutVisualCount }: { index: number; cutVisualCount: number }) {
  const isCut = index < cutVisualCount;
  const cutIndex = cutVisualCount - index - 1;
  const mainIndex = index - cutVisualCount;
  const mainDepth = Math.min(mainIndex, 13);
  const cutDepth = Math.min(cutIndex, 13);
  const transform = isCut
    ? `translate3d(${22 + cutDepth * 1.5}px, ${30 - cutDepth * 0.45}px, 0) rotate(${-7 + cutDepth * 0.5}deg)`
    : `translate3d(${132 + mainDepth * 1.3}px, ${18 + mainDepth * 1.2}px, 0) rotate(${0.5 + mainDepth * 0.04}deg)`;

  return (
    <div
      className={cn(
        "absolute size-[4.25rem] rounded-md border shadow-sm transition-[transform,opacity,box-shadow] duration-300 ease-out",
        isCut
          ? "border-primary/55 bg-primary/10 shadow-md"
          : "border-border bg-card shadow-foreground/5",
      )}
      style={{
        transform,
        zIndex: isCut ? 90 + cutIndex : VISUAL_CARD_COUNT - mainIndex,
        opacity: isCut || mainIndex < 20 ? 1 : 0,
        transitionDelay: `${isCut ? cutIndex * 10 : mainIndex * 8}ms`,
      }}
    >
      <div className={cn("m-2 h-1.5 rounded-full", isCut ? "bg-primary/35" : "bg-muted")} />
      <div
        className={cn("mx-2 mt-8 h-1.5 rounded-full", isCut ? "bg-primary/20" : "bg-muted/70")}
      />
    </div>
  );
}

function CutDeckControl({
  cutSize,
  clampedCutSize,
  maxCutSize,
  onCutSizeChange,
}: {
  cutSize: string;
  clampedCutSize: number;
  maxCutSize: number;
  onCutSizeChange: (cutSize: string) => void;
}) {
  const remainingCards = maxCutSize - clampedCutSize;
  const cutVisualCount =
    maxCutSize > 0 ? Math.round((clampedCutSize / maxCutSize) * VISUAL_CARD_COUNT) : 0;

  return (
    <div className="grid gap-4">
      <div className="grid gap-3 rounded-lg border border-border/70 bg-background p-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Label htmlFor="cut-size" className="text-xs font-medium uppercase tracking-[0.16em]">
            Cut size
          </Label>
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{clampedCutSize} cut</Badge>
            <Badge variant="outline">{remainingCards} deck</Badge>
          </div>
        </div>

        <div className="relative h-40 overflow-hidden rounded-lg border border-border/70 bg-muted/20">
          <div className="absolute inset-x-4 top-4 flex justify-between text-[0.68rem] uppercase tracking-[0.14em] text-muted-foreground">
            <span>Cut packet</span>
            <span>Dealing deck</span>
          </div>
          <div className="absolute left-2 top-12 h-24 w-[15.75rem] sm:left-1/2 sm:-translate-x-1/2">
            {Array.from({ length: VISUAL_CARD_COUNT }, (_, index) => (
              <DeckCard key={index} index={index} cutVisualCount={cutVisualCount} />
            ))}
          </div>
        </div>

        <Input
          id="cut-size"
          type="range"
          min={0}
          max={maxCutSize}
          value={clampedCutSize}
          onChange={(event) => onCutSizeChange(event.target.value)}
          className="h-2 cursor-pointer p-0 accent-primary"
        />

        <div className="grid grid-cols-[minmax(0,1fr)_5rem] items-center gap-2">
          <p className="text-xs text-muted-foreground">
            Main {remainingCards} / packet {clampedCutSize}
          </p>
          <Input
            type="number"
            inputMode="numeric"
            min={0}
            max={maxCutSize}
            value={cutSize}
            onChange={(event) => onCutSizeChange(event.target.value)}
            aria-label="Exact cut size"
            className="h-9 text-right"
          />
        </div>
      </div>
    </div>
  );
}

function SortableDealOrderPlayer({
  player,
  playerIndex,
  position,
}: {
  player: PlayerSnapshot;
  playerIndex: number;
  position: number;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: dealPlayerId(playerIndex),
  });
  const style = {
    transform: transform
      ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0)`
      : undefined,
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex min-h-14 items-center gap-3 rounded-lg border border-border/70 bg-background px-3 py-2 text-sm shadow-sm",
        isDragging ? "opacity-60" : undefined,
      )}
      {...attributes}
      {...listeners}
    >
      <Badge variant="outline" className="w-9 justify-center">
        {position + 1}
      </Badge>
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium">{player.name}</p>
        <p className="text-xs text-muted-foreground">Seat {player.seat + 1}</p>
      </div>
      <HugeiconsIcon
        icon={DragDropVerticalIcon}
        strokeWidth={2}
        className="text-muted-foreground"
      />
    </div>
  );
}

function StepMarker({ step, active, done }: { step: number; active: boolean; done: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <span
        className={cn(
          "flex size-7 items-center justify-center rounded-full border text-xs font-semibold",
          active || done
            ? "border-primary bg-primary text-primary-foreground"
            : "border-border bg-background text-muted-foreground",
        )}
      >
        {step}
      </span>
      <span
        className={cn("text-sm font-medium", active ? "text-foreground" : "text-muted-foreground")}
      >
        {step === 1 ? "Cut" : "Deal"}
      </span>
    </div>
  );
}

export function DealChoicePanel({
  players,
  pendingDealChoice,
  dealChooserName,
  isDealChooser,
  onChooseDealing,
}: DealChoicePanelProps) {
  const [dealStep, setDealStep] = useState<DealStep>("cut");
  const [dealMode, setDealMode] = useState<DealMode>("round_robin");
  const [cutSize, setCutSize] = useState("0");
  const [tapOrder, setTapOrder] = useState<number[]>([]);
  const dealChoiceKey = [
    pendingDealChoice.dealerIndex,
    pendingDealChoice.chooserIndex,
    players.map((player) => player.playerId).join(","),
  ].join(":");
  const maxCutSize = Math.max(0, GAME_DECK_CARD_COUNT - players.length * 12);
  const clampedCutSize = clampCutSize(cutSize, maxCutSize);
  const dealerName = players[pendingDealChoice.dealerIndex]?.name ?? null;
  const tapOrderIds = tapOrder.map(dealPlayerId);

  useEffect(() => {
    setDealStep("cut");
    setDealMode("round_robin");
    setCutSize("0");
    setTapOrder(defaultTapOrder(players, pendingDealChoice));
  }, [dealChoiceKey, pendingDealChoice, players]);

  function handleTapOrderDragEnd(event: DragEndEvent) {
    const { active, over } = event;

    if (!over || active.id === over.id) {
      return;
    }

    const activeIndex = playerIndexFromDealId(String(active.id));
    const overIndex = playerIndexFromDealId(String(over.id));
    const oldIndex = tapOrder.indexOf(activeIndex);
    const newIndex = tapOrder.indexOf(overIndex);

    if (oldIndex < 0 || newIndex < 0) {
      return;
    }

    setTapOrder((currentOrder) => arrayMove(currentOrder, oldIndex, newIndex));
  }

  function handleChooseDealing() {
    if (dealMode === "tap") {
      onChooseDealing({
        dealType: "tap",
        cutSize: clampedCutSize,
        order: tapOrder,
      });
      return;
    }

    onChooseDealing({
      dealType: "round_robin",
      cutSize: clampedCutSize,
    });
  }

  return (
    <div className="grid gap-4 rounded-lg border border-primary/30 bg-primary/5 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <p className="text-sm font-medium">Deal choice</p>
          <p className="text-xs text-muted-foreground">
            {dealerName ? `Dealer: ${dealerName}` : "Dealer selected"}
          </p>
        </div>
        <Badge variant={isDealChooser ? "default" : "outline"}>
          {isDealChooser ? "Your pick" : (dealChooserName ?? "Waiting")}
        </Badge>
      </div>

      {isDealChooser ? (
        <div className="grid gap-4">
          <div className="flex items-center gap-3 rounded-lg border border-border/70 bg-background p-2">
            <StepMarker step={1} active={dealStep === "cut"} done={dealStep === "deal"} />
            <div className="h-px min-w-6 flex-1 bg-border" />
            <StepMarker step={2} active={dealStep === "deal"} done={false} />
          </div>

          {dealStep === "cut" ? (
            <div className="grid gap-4">
              <CutDeckControl
                cutSize={cutSize}
                clampedCutSize={clampedCutSize}
                maxCutSize={maxCutSize}
                onCutSizeChange={setCutSize}
              />
              <Button type="button" onClick={() => setDealStep("deal")}>
                Continue
                <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} data-icon="inline-end" />
              </Button>
            </div>
          ) : (
            <div className="grid gap-4">
              <div className="grid gap-2">
                <p className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
                  Dealing style
                </p>
                <div className="grid grid-cols-2 gap-2">
                  <Button
                    type="button"
                    variant={dealMode === "round_robin" ? "default" : "outline"}
                    onClick={() => setDealMode("round_robin")}
                  >
                    <HugeiconsIcon icon={Cards02Icon} strokeWidth={2} data-icon="inline-start" />
                    Round robin
                  </Button>
                  <Button
                    type="button"
                    variant={dealMode === "tap" ? "default" : "outline"}
                    onClick={() => setDealMode("tap")}
                  >
                    <HugeiconsIcon icon={ArrangeIcon} strokeWidth={2} data-icon="inline-start" />
                    Tap order
                  </Button>
                </div>
              </div>

              {dealMode === "tap" ? (
                <div className="grid gap-2">
                  <p className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
                    Dealing order
                  </p>
                  <DndContext collisionDetection={closestCenter} onDragEnd={handleTapOrderDragEnd}>
                    <SortableContext items={tapOrderIds} strategy={verticalListSortingStrategy}>
                      <div className="grid gap-2">
                        {tapOrder.map((playerIndex, index) => {
                          const player = players[playerIndex];

                          if (!player) {
                            return null;
                          }

                          return (
                            <SortableDealOrderPlayer
                              key={player.playerId}
                              player={player}
                              playerIndex={playerIndex}
                              position={index}
                            />
                          );
                        })}
                      </div>
                    </SortableContext>
                  </DndContext>
                </div>
              ) : null}

              <div className="grid grid-cols-[auto_minmax(0,1fr)] gap-2">
                <Button type="button" variant="outline" onClick={() => setDealStep("cut")}>
                  <HugeiconsIcon icon={ArrowLeft02Icon} strokeWidth={2} data-icon="inline-start" />
                  Back
                </Button>
                <Button type="button" onClick={handleChooseDealing}>
                  Start round
                </Button>
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="grid gap-1 text-sm text-muted-foreground">
          <p>
            Waiting for{" "}
            <span className="font-medium text-foreground">
              {dealChooserName ?? "the deal chooser"}
            </span>{" "}
            to cut the deck and choose the dealing style.
          </p>
        </div>
      )}
    </div>
  );
}

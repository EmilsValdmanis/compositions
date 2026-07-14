import { useEffect, useEffectEvent, useState } from "react";
import { closestCenter, DndContext, type DragEndEvent, type Modifier } from "@dnd-kit/core";
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
import { motion, useReducedMotion } from "motion/react";
import {
  type DealingChoiceRequest,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
} from "#/components/game-websocket-provider";
import { FACE_DOWN_CARD } from "#/components/game/game-board-view-state";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "#/components/ui/field";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "#/components/ui/item";
import { Slider } from "#/components/ui/slider";
import { ToggleGroup, ToggleGroupItem } from "#/components/ui/toggle-group";
import { cn } from "#/lib/utils";

const GAME_DECK_CARD_COUNT = 108;
const DECK_STACK_BASE_TOP = 4;
const DECK_CARD_VERTICAL_OFFSET = 0.25;

type DealMode = "round_robin" | "tap";
type DealStep = "cut" | "deal";

const restrictTapOrderDrag: Modifier = ({ activeNodeRect, containerNodeRect, transform }) => {
  if (!activeNodeRect || !containerNodeRect) {
    return { ...transform, x: 0 };
  }

  const minY = containerNodeRect.top - activeNodeRect.top;
  const maxY = containerNodeRect.bottom - activeNodeRect.bottom;

  return {
    ...transform,
    x: 0,
    y: Math.min(Math.max(transform.y, minY), maxY),
  };
};

type DealChoicePanelProps = {
  players: PlayerSnapshot[];
  pendingDealChoice: PendingDealChoiceSnapshot;
  dealChooserName: string | null;
  isDealChooser: boolean;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
};

function defaultTapOrder(players: PlayerSnapshot[], pendingDealChoice: PendingDealChoiceSnapshot) {
  const activeIndexes = players.flatMap((player, index) => (player.forfeited ? [] : [index]));
  const dealerPosition = activeIndexes.indexOf(pendingDealChoice.dealerIndex);
  if (dealerPosition < 0) {
    return activeIndexes;
  }
  return activeIndexes.map(
    (_, offset) => activeIndexes[(dealerPosition + offset + 1) % activeIndexes.length]!,
  );
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
    transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined,
    transition,
  };

  return (
    <Item
      ref={setNodeRef}
      style={style}
      variant="outline"
      size="sm"
      className={cn(
        "min-w-0 cursor-grab touch-none active:cursor-grabbing",
        isDragging && "opacity-60",
      )}
      {...attributes}
      {...listeners}
    >
      <ItemMedia>
        <Badge variant="secondary">{position + 1}</Badge>
      </ItemMedia>
      <ItemContent>
        <ItemTitle>{player.name}</ItemTitle>
        <ItemDescription>Seat {player.seat + 1}</ItemDescription>
      </ItemContent>
      <ItemActions>
        <HugeiconsIcon icon={DragDropVerticalIcon} strokeWidth={2} />
      </ItemActions>
    </Item>
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
    players.map((player) => `${player.playerId}:${Boolean(player.forfeited)}`).join(","),
  ].join(":");
  const activePlayerCount = players.filter((player) => !player.forfeited).length;
  const maxCutSize = Math.max(0, GAME_DECK_CARD_COUNT - activePlayerCount * 12);
  const clampedCutSize = clampCutSize(cutSize, maxCutSize);
  const prefersReducedMotion = useReducedMotion();
  const dealerName = players[pendingDealChoice.dealerIndex]?.name ?? null;
  const tapOrderIds = tapOrder.map(dealPlayerId);
  const resetDealChoice = useEffectEvent(() => {
    setDealStep("cut");
    setDealMode("round_robin");
    setCutSize("0");
    setTapOrder(defaultTapOrder(players, pendingDealChoice));
  });

  useEffect(() => {
    resetDealChoice();
  }, [dealChoiceKey]);

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
    <Card size="sm" className="w-full">
      <CardHeader>
        <CardTitle>Prepare the deal</CardTitle>
        <CardDescription>
          {dealerName ? `Dealer: ${dealerName}` : "Dealer selected"}
        </CardDescription>
        <CardAction>
          <Badge variant={isDealChooser ? "secondary" : "outline"}>
            {isDealChooser ? `Step ${dealStep === "cut" ? 1 : 2} of 2` : "Waiting"}
          </Badge>
        </CardAction>
      </CardHeader>

      <CardContent>
        {isDealChooser ? (
          dealStep === "cut" ? (
            <FieldGroup>
              <Item variant="muted">
                <ItemMedia variant="icon">
                  <HugeiconsIcon icon={Cards02Icon} strokeWidth={2} />
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>Cut the deck</ItemTitle>
                  <ItemDescription>Choose how many cards to lift before dealing.</ItemDescription>
                </ItemContent>
                <ItemFooter>
                  <div className="w-full">
                    <div
                      className="relative mx-auto h-32 w-full max-w-sm overflow-hidden perspective-[520px]"
                      aria-hidden="true"
                    >
                      <motion.div
                        initial={false}
                        animate={{
                          left: "25%",
                          x: "-50%",
                          opacity: clampedCutSize > 0 ? 0.55 : 0,
                          scaleX: 0.65 + (clampedCutSize / GAME_DECK_CARD_COUNT) * 0.35,
                        }}
                        transition={{ duration: prefersReducedMotion ? 0 : 0.2 }}
                        className="absolute bottom-7 h-2 w-20 rounded-full bg-black/40 blur-md"
                      />
                      <motion.div
                        initial={false}
                        animate={{
                          left: "75%",
                          x: "-50%",
                          opacity: 0.55,
                          scaleX:
                            0.65 +
                            ((GAME_DECK_CARD_COUNT - clampedCutSize) / GAME_DECK_CARD_COUNT) * 0.35,
                        }}
                        transition={{ duration: prefersReducedMotion ? 0 : 0.2 }}
                        className="absolute bottom-7 h-2 w-20 rounded-full bg-black/40 blur-md"
                      />
                      {Array.from({ length: GAME_DECK_CARD_COUNT }, (_, index) => {
                        const firstLiftedIndex = GAME_DECK_CARD_COUNT - clampedCutSize;
                        const isLifted = index >= firstLiftedIndex;
                        const stackIndex = isLifted ? index - firstLiftedIndex : index;
                        const stackSize = isLifted
                          ? clampedCutSize
                          : GAME_DECK_CARD_COUNT - clampedCutSize;
                        const depth = Math.max(0, stackSize - stackIndex - 1);

                        return (
                          <motion.div
                            key={index}
                            initial={false}
                            animate={{
                              left: isLifted ? "25%" : "75%",
                              x: "-50%",
                              top: DECK_STACK_BASE_TOP - stackIndex * DECK_CARD_VERTICAL_OFFSET,
                              rotateX: 54,
                              rotateZ: isLifted ? -5 + (index % 3) * 0.4 : 4 - (index % 3) * 0.35,
                            }}
                            transition={
                              prefersReducedMotion
                                ? { duration: 0 }
                                : { type: "spring", stiffness: 230, damping: 25, mass: 0.75 }
                            }
                            className="absolute h-24 w-17 origin-bottom"
                            style={{
                              zIndex: isLifted
                                ? GAME_DECK_CARD_COUNT + stackIndex + 1
                                : stackIndex + 1,
                            }}
                          >
                            <GameCard
                              card={FACE_DOWN_CARD}
                              faceDown
                              size="default"
                              className={cn("size-full", depth === 0 ? "shadow-lg" : "shadow-none")}
                            />
                          </motion.div>
                        );
                      })}
                    </div>
                    <div className="mx-auto grid w-full max-w-sm grid-cols-2 gap-6">
                      <div className="flex justify-center">
                        <Badge variant="secondary">Lifted · {clampedCutSize}</Badge>
                      </div>
                      <div className="flex justify-center">
                        <Badge variant="secondary">
                          Deck · {GAME_DECK_CARD_COUNT - clampedCutSize}
                        </Badge>
                      </div>
                    </div>
                  </div>
                </ItemFooter>
              </Item>

              <Field>
                <FieldLabel id="cut-size-label" htmlFor="cut-size" className="sr-only">
                  Cut size
                </FieldLabel>
                <Slider
                  id="cut-size"
                  min={0}
                  max={maxCutSize}
                  step={1}
                  thumbAriaLabelledBy="cut-size-label"
                  value={[clampedCutSize]}
                  onValueChange={(value) => {
                    const nextCutSize = Array.isArray(value) ? value[0] : value;
                    setCutSize(String(nextCutSize ?? 0));
                  }}
                />
                <FieldDescription className="flex justify-between">
                  <span>0</span>
                  <span>{maxCutSize} cards maximum</span>
                </FieldDescription>
              </Field>
            </FieldGroup>
          ) : (
            <FieldGroup>
              <FieldSet>
                <FieldLegend variant="label">Dealing style</FieldLegend>
                <ToggleGroup
                  value={[dealMode]}
                  onValueChange={(value) => {
                    const nextMode = value[0];
                    if (nextMode === "round_robin" || nextMode === "tap") {
                      setDealMode(nextMode);
                    }
                  }}
                  variant="outline"
                  spacing={2}
                  className="grid w-full grid-cols-2"
                >
                  <ToggleGroupItem value="round_robin" className="min-w-0">
                    <HugeiconsIcon icon={Cards02Icon} strokeWidth={2} />
                    Round robin
                  </ToggleGroupItem>
                  <ToggleGroupItem value="tap" className="min-w-0">
                    <HugeiconsIcon icon={ArrangeIcon} strokeWidth={2} />
                    Tap order
                  </ToggleGroupItem>
                </ToggleGroup>
                <FieldDescription>
                  {dealMode === "round_robin"
                    ? "Deal one card to each player in turn."
                    : "Deal each player’s full hand in the order below."}
                </FieldDescription>
              </FieldSet>

              {dealMode === "tap" ? (
                <FieldSet>
                  <FieldLegend variant="label">Player order</FieldLegend>
                  <DndContext
                    autoScroll={false}
                    collisionDetection={closestCenter}
                    modifiers={[restrictTapOrderDrag]}
                    onDragEnd={handleTapOrderDragEnd}
                  >
                    <SortableContext items={tapOrderIds} strategy={verticalListSortingStrategy}>
                      <ItemGroup className="max-h-56 overflow-y-auto pr-1">
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
                      </ItemGroup>
                    </SortableContext>
                  </DndContext>
                </FieldSet>
              ) : null}
            </FieldGroup>
          )
        ) : (
          <Item variant="muted">
            <ItemMedia variant="icon">
              <HugeiconsIcon icon={Cards02Icon} strokeWidth={2} />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>Waiting for {dealChooserName ?? "the deal chooser"}</ItemTitle>
              <ItemDescription>They’re cutting the deck and choosing how to deal.</ItemDescription>
            </ItemContent>
          </Item>
        )}
      </CardContent>

      {isDealChooser ? (
        <CardFooter className="gap-2">
          {dealStep === "deal" ? (
            <Button type="button" variant="outline" onClick={() => setDealStep("cut")}>
              <HugeiconsIcon icon={ArrowLeft02Icon} strokeWidth={2} data-icon="inline-start" />
              Back
            </Button>
          ) : null}
          <Button
            type="button"
            className="flex-1"
            onClick={() => {
              if (dealStep === "cut") {
                setDealStep("deal");
                return;
              }
              handleChooseDealing();
            }}
          >
            {dealStep === "cut" ? "Continue" : "Start round"}
            {dealStep === "cut" ? (
              <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} data-icon="inline-end" />
            ) : null}
          </Button>
        </CardFooter>
      ) : null}
    </Card>
  );
}

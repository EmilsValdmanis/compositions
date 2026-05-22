import { type CSSProperties, type ReactNode } from "react";
import { useDraggable } from "@dnd-kit/core";
import { useSortable } from "@dnd-kit/sortable";
import {
  Clubs02Icon,
  Diamond01Icon,
  FavouriteIcon,
  JokerIcon,
  SpadesIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { type CardSnapshot } from "#/components/game-websocket-provider";
import { cardName } from "#/components/game/game-card-utils";
import { cn } from "#/lib/utils";

const rankLabels: Record<number, string> = {
  1: "A",
  2: "2",
  3: "3",
  4: "4",
  5: "5",
  6: "6",
  7: "7",
  8: "8",
  9: "9",
  10: "10",
  11: "J",
  12: "Q",
  13: "K",
};

const suitIcons = {
  0: FavouriteIcon,
  1: Diamond01Icon,
  2: Clubs02Icon,
  3: SpadesIcon,
};

function cardRankLabel(card: CardSnapshot) {
  if (card.isJoker) {
    return "J";
  }

  return rankLabels[card.rank ?? 0] ?? "?";
}

function cardSuitIcon(card: CardSnapshot) {
  if (card.isJoker) {
    return JokerIcon;
  }

  return suitIcons[card.suit as keyof typeof suitIcons] ?? null;
}

function cardAccentClass(card: CardSnapshot) {
  if (card.isJoker) {
    return "border-primary/40 bg-primary/10 text-primary";
  }

  if (card.suit === 0 || card.suit === 1) {
    return "border-destructive/30 bg-destructive/5 text-destructive";
  }

  return "border-foreground/15 bg-background text-foreground";
}

type GameCardSize = "hand" | "default" | "compact";

const gameCardSizeClassNames: Record<GameCardSize, string> = {
  hand: "h-32 w-20 p-2.5",
  default: "h-24 w-16 p-2",
  compact: "h-16 w-11 rounded-xl p-1.5",
};

const cardCornerTextClassNames: Record<GameCardSize, string> = {
  hand: "text-lg",
  default: "text-sm",
  compact: "text-[0.65rem]",
};

const cardCornerInsetClassNames: Record<GameCardSize, string> = {
  hand: "left-2.5 top-2.5",
  default: "left-2 top-2",
  compact: "left-1.5 top-1.5",
};

const cardCornerEndInsetClassNames: Record<GameCardSize, string> = {
  hand: "bottom-2.5 right-2.5",
  default: "bottom-2 right-2",
  compact: "bottom-1.5 right-1.5",
};

const cardSymbolClassNames: Record<GameCardSize, string> = {
  hand: "size-8",
  default: "size-6",
  compact: "size-4",
};

function gameCardClassName(card: CardSnapshot, size: GameCardSize, className?: string) {
  return cn(
    "relative grid shrink-0 select-none place-items-center rounded-2xl border shadow-sm transition",
    cardAccentClass(card),
    gameCardSizeClassNames[size],
    className,
  );
}

function faceDownGameCardClassName(size: GameCardSize, className?: string) {
  return cn(
    "relative grid shrink-0 select-none place-items-center rounded-2xl border border-foreground/10 bg-foreground/5 shadow-sm transition",
    gameCardSizeClassNames[size],
    className,
  );
}

function renderGameCardFace(card: CardSnapshot, size: GameCardSize) {
  const rank = cardRankLabel(card);
  const icon = cardSuitIcon(card);

  return (
    <>
      <span
        className={cn(
          "absolute leading-none font-semibold",
          cardCornerInsetClassNames[size],
          cardCornerTextClassNames[size],
        )}
      >
        {rank}
      </span>
      {icon ? (
        <HugeiconsIcon
          icon={icon}
          className={cn("pointer-events-none", cardSymbolClassNames[size])}
          aria-hidden="true"
        />
      ) : (
        <span className={cn("pointer-events-none leading-none", cardSymbolClassNames[size])}>
          ?
        </span>
      )}
      <span
        className={cn(
          "absolute leading-none font-semibold",
          cardCornerEndInsetClassNames[size],
          cardCornerTextClassNames[size],
        )}
      >
        {rank}
      </span>
    </>
  );
}

function renderGameCardBack() {
  return <span className="absolute inset-1.5 rounded-xl border border-white/10" />;
}

type GameCardDecoration = {
  label?: ReactNode;
  footer?: ReactNode;
  highlight?: "new" | "addition" | "joker_reclaim";
};

function decorationRingClassName(highlight?: GameCardDecoration["highlight"]) {
  switch (highlight) {
    case "new":
      return "border-primary/60 ring-1 ring-primary/60 ring-offset-2 ring-offset-background shadow-[0_0_0_1px_hsl(var(--primary)/0.2)]";
    case "addition":
      return "border-primary/70 ring-2 ring-primary ring-offset-2 ring-offset-background shadow-[0_0_0_1px_hsl(var(--primary)/0.35)]";
    case "joker_reclaim":
      return "border-primary/70 ring-2 ring-primary ring-offset-2 ring-offset-background shadow-[0_0_0_1px_hsl(var(--primary)/0.35)]";
    default:
      return null;
  }
}

function GameCardDecorationLayer({
  size,
  decoration,
}: {
  size: GameCardSize;
  decoration?: GameCardDecoration;
}) {
  if (!decoration?.label && !decoration?.footer) {
    return null;
  }

  return (
    <>
      {decoration.label ? (
        <div
          className={cn(
            "absolute bottom-0 left-1/2 z-10 -translate-x-1/2 translate-y-[calc(100%+0.16rem)] whitespace-nowrap",
            size === "hand" ? "text-[0.7rem]" : null,
          )}
        >
          {decoration.label}
        </div>
      ) : null}
      {decoration.footer ? (
        <div className="absolute bottom-1 left-1/2 z-10 flex -translate-x-1/2 translate-y-1/2 items-center gap-1 rounded-full bg-background/90 px-1 py-0.5 shadow-sm backdrop-blur-sm">
          {decoration.footer}
        </div>
      ) : null}
    </>
  );
}

function SortableGameCard({
  card,
  id,
  cardIndex,
  size,
  className,
  faceDown,
  data,
  decoration,
}: {
  card: CardSnapshot;
  id: string;
  cardIndex: number;
  size: GameCardSize;
  className?: string;
  faceDown?: boolean;
  data?: Record<string, unknown>;
  decoration?: GameCardDecoration;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { cardIndex, ...data },
  });
  const accessibleName = faceDown ? "Face-down card" : cardName(card);

  const style: CSSProperties = {
    transform: transform
      ? `translate3d(${transform.x}px, ${transform.y}px, 0) scaleX(${transform.scaleX}) scaleY(${transform.scaleY})`
      : undefined,
    transition,
    opacity: isDragging ? 0 : 1,
    zIndex: isDragging ? 20 : undefined,
  };

  return (
    <button
      ref={setNodeRef}
      type="button"
      style={style}
      className={cn(
        faceDown
          ? faceDownGameCardClassName(size, className)
          : gameCardClassName(card, size, className),
        decorationRingClassName(decoration?.highlight),
        "touch-none cursor-grab active:cursor-grabbing",
      )}
      title={accessibleName}
      aria-label={accessibleName}
      {...listeners}
      {...attributes}
    >
      {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
      <GameCardDecorationLayer size={size} decoration={decoration} />
    </button>
  );
}

function DraggableGameCard({
  card,
  id,
  size,
  className,
  disabled,
  faceDown,
  data,
  decoration,
}: {
  card: CardSnapshot;
  id: string;
  size: GameCardSize;
  className?: string;
  disabled?: boolean;
  faceDown?: boolean;
  data?: Record<string, unknown>;
  decoration?: GameCardDecoration;
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id,
    disabled,
    data,
  });
  const accessibleName = faceDown ? "Face-down card" : cardName(card);

  const style: CSSProperties = {
    transform: transform ? `translate3d(${transform.x}px, ${transform.y}px, 0)` : undefined,
    opacity: isDragging ? 0 : 1,
    zIndex: isDragging ? 20 : undefined,
  };

  return (
    <button
      ref={setNodeRef}
      type="button"
      style={style}
      disabled={disabled}
      className={cn(
        faceDown
          ? faceDownGameCardClassName(size, className)
          : gameCardClassName(card, size, className),
        decorationRingClassName(decoration?.highlight),
        disabled ? "cursor-default opacity-50" : "touch-none cursor-grab active:cursor-grabbing",
      )}
      title={accessibleName}
      aria-label={accessibleName}
      {...listeners}
      {...attributes}
    >
      {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
      <GameCardDecorationLayer size={size} decoration={decoration} />
    </button>
  );
}

export function GameCard({
  card,
  size = "default",
  className,
  draggable,
  dragSource,
  faceDown,
  decoration,
}: {
  card: CardSnapshot;
  size?: GameCardSize;
  className?: string;
  draggable?: {
    id: string;
    cardIndex: number;
    isVirtual?: boolean;
  };
  dragSource?: {
    id: string;
    disabled?: boolean;
    data?: Record<string, unknown>;
  };
  faceDown?: boolean;
  decoration?: GameCardDecoration;
}) {
  if (draggable) {
    return (
      <SortableGameCard
        card={card}
        id={draggable.id}
        cardIndex={draggable.cardIndex}
        size={size}
        className={className}
        faceDown={faceDown}
        data={{ isVirtual: draggable.isVirtual }}
        decoration={decoration}
      />
    );
  }

  if (dragSource) {
    return (
      <DraggableGameCard
        card={card}
        id={dragSource.id}
        size={size}
        className={className}
        disabled={dragSource.disabled}
        data={dragSource.data}
        faceDown={faceDown}
        decoration={decoration}
      />
    );
  }

  const accessibleName = faceDown ? "Face-down card" : cardName(card);

  return (
    <div
      className={
        faceDown
          ? faceDownGameCardClassName(
              size,
              cn(className, decorationRingClassName(decoration?.highlight)),
            )
          : gameCardClassName(
              card,
              size,
              cn(className, decorationRingClassName(decoration?.highlight)),
            )
      }
      title={accessibleName}
      aria-label={accessibleName}
    >
      {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
      <GameCardDecorationLayer size={size} decoration={decoration} />
    </div>
  );
}

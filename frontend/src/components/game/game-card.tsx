import { type CSSProperties, type ReactNode } from "react";
import { useDraggable } from "@dnd-kit/core";
import { useSortable } from "@dnd-kit/sortable";
import { motion, type Variants } from "motion/react";
import {
  Clubs02Icon,
  Diamond01Icon,
  FavouriteIcon,
  JokerIcon,
  SpadesIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react";

import { type CardSnapshot } from "#/components/game-websocket-provider";
import { cardName } from "#/components/game/game-card-utils";
import { cn } from "#/lib/utils";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { m } from "#/paraglide/messages.js";

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
    return "border-primary/40 bg-card text-primary";
  }

  if (card.suit === 0 || card.suit === 1) {
    return "border-border bg-card text-destructive";
  }

  return "border-border bg-card text-foreground";
}

type GameCardSize = "hand" | "default" | "compact";

const gameCardSizeClassNames: Record<GameCardSize, string> = {
  hand: "h-20 w-14 p-1.5 xl:h-32 xl:w-20 xl:p-2.5",
  default: "h-20 w-14 p-1.5 xl:h-24 xl:w-16 xl:p-2",
  compact: "h-16 w-11 rounded-lg p-1.5",
};

const gameCardFrameSizeClassNames: Record<GameCardSize, string> = {
  hand: "h-20 w-14 xl:h-32 xl:w-20",
  default: "h-20 w-14 xl:h-24 xl:w-16",
  compact: "h-16 w-11",
};

const cardMotionVariants: Variants = {
  rest: (shouldReduceMotion: boolean) => ({
    transform: shouldReduceMotion ? "none" : "translateY(0px) scale(1)",
    filter: "drop-shadow(0 0 0 rgb(0 0 0 / 0))",
    transition: {
      transform: { type: "spring", duration: 0.32, bounce: 0.22 },
      filter: { duration: 0.14, ease: [0.23, 1, 0.32, 1] },
    },
  }),
  hover: (shouldReduceMotion: boolean) => ({
    transform: shouldReduceMotion ? "none" : "translateY(-5px) scale(1)",
    filter: "drop-shadow(0 8px 7px rgb(0 0 0 / 0.22))",
    transition: {
      transform: { type: "spring", stiffness: 500, damping: 26, mass: 0.65 },
      filter: { duration: 0.16, ease: [0.23, 1, 0.32, 1] },
    },
  }),
  pressed: (shouldReduceMotion: boolean) => ({
    transform: shouldReduceMotion ? "none" : "translateY(-8px) scale(1.015)",
    filter: "drop-shadow(0 16px 12px rgb(0 0 0 / 0.32))",
    transition: {
      transform: { type: "spring", stiffness: 700, damping: 32, mass: 0.55 },
      filter: { duration: 0.12, ease: [0.23, 1, 0.32, 1] },
    },
  }),
};

const cardCornerTextClassNames: Record<GameCardSize, string> = {
  hand: "text-xs/none xl:text-lg/none",
  default: "text-xs/none xl:text-sm/none",
  compact: "text-[0.65rem]/none",
};

const cardCornerStartInsetClassNames: Record<GameCardSize, string> = {
  hand: "left-1.5 top-1.5 xl:left-2.5 xl:top-2.5",
  default: "left-1.5 top-1.5 xl:left-2 xl:top-2",
  compact: "left-1.5 top-1.5",
};

const cardCornerEndInsetClassNames: Record<GameCardSize, string> = {
  hand: "bottom-1.5 right-1.5 xl:bottom-2.5 xl:right-2.5",
  default: "bottom-1.5 right-1.5 xl:bottom-2 xl:right-2",
  compact: "bottom-1.5 right-1.5",
};

const cardCenterSuitClassNames: Record<GameCardSize, string> = {
  hand: "size-6 xl:size-8",
  default: "size-6 xl:size-7",
  compact: "size-[1.125rem]",
};

function gameCardClassName(card: CardSnapshot, size: GameCardSize, className?: string) {
  return cn(
    "relative grid shrink-0 select-none place-items-center overflow-hidden rounded-xl border shadow-sm transition-[transform,box-shadow,opacity] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)]",
    cardAccentClass(card),
    gameCardSizeClassNames[size],
    className,
  );
}

function faceDownGameCardClassName(size: GameCardSize, className?: string) {
  return cn(
    "relative grid shrink-0 select-none place-items-center overflow-hidden rounded-xl border border-border bg-card shadow-sm transition-[transform,box-shadow,opacity] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)]",
    gameCardSizeClassNames[size],
    className,
  );
}

function OutlinedCardIcon({ icon, className }: { icon: IconSvgElement; className?: string }) {
  return (
    <HugeiconsIcon
      icon={icon}
      className={cn("pointer-events-none", className)}
      strokeWidth={1.5}
      aria-hidden="true"
    />
  );
}

function CardCorner({ rank, size, end }: { rank: string; size: GameCardSize; end?: boolean }) {
  return (
    <span
      className={cn(
        "absolute z-10 font-semibold tracking-[-0.04em]",
        cardCornerTextClassNames[size],
        end ? cardCornerEndInsetClassNames[size] : cardCornerStartInsetClassNames[size],
      )}
    >
      {rank}
    </span>
  );
}

function JokerCardArt({ size }: { size: GameCardSize }) {
  return (
    <span className="pointer-events-none absolute inset-[18%] grid place-items-center">
      <OutlinedCardIcon icon={JokerIcon} className={cardCenterSuitClassNames[size]} />
    </span>
  );
}

function renderGameCardFace(card: CardSnapshot, size: GameCardSize) {
  const rankLabel = cardRankLabel(card);
  const suitIcon = cardSuitIcon(card);

  return (
    <>
      <CardCorner rank={rankLabel} size={size} />
      {card.isJoker ? (
        <JokerCardArt size={size} />
      ) : suitIcon && card.rank ? (
        <OutlinedCardIcon icon={suitIcon} className={cardCenterSuitClassNames[size]} />
      ) : (
        <span
          className={cn(
            "pointer-events-none text-center font-semibold",
            cardCornerTextClassNames[size],
          )}
        >
          ?
        </span>
      )}
      <CardCorner rank={rankLabel} size={size} end />
    </>
  );
}

function renderGameCardBack() {
  return <span className="absolute inset-1.5 rounded-xl border border-foreground/10 bg-muted" />;
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

function invalidRingClassName(invalid?: boolean) {
  return invalid
    ? "border-destructive ring-2 ring-destructive ring-offset-2 ring-offset-background shadow-[0_0_0_1px_hsl(var(--destructive)/0.35)]"
    : null;
}

function GameCardDecorationLayer({ decoration }: { decoration?: GameCardDecoration }) {
  if (!decoration?.label && !decoration?.footer) {
    return null;
  }

  return (
    <>
      {decoration.label ? (
        <div className="absolute bottom-0 left-1/2 z-10 -translate-x-1/2 translate-y-[calc(100%+0.16rem)] whitespace-nowrap">
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
  invalid,
}: {
  card: CardSnapshot;
  id: string;
  cardIndex: number;
  size: GameCardSize;
  className?: string;
  faceDown?: boolean;
  data?: Record<string, unknown>;
  decoration?: GameCardDecoration;
  invalid?: boolean;
}) {
  const shouldReduceMotion = useShouldReduceMotion();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { cardIndex, ...data },
  });
  const accessibleName = faceDown ? m.face_down_card() : cardName(card);

  const style: CSSProperties = {
    transform: transform
      ? `translate3d(${transform.x}px, ${transform.y}px, 0) scaleX(${transform.scaleX}) scaleY(${transform.scaleY})`
      : undefined,
    transition: shouldReduceMotion ? undefined : transition,
    opacity: isDragging ? 0 : 1,
    zIndex: isDragging ? 20 : undefined,
  };

  return (
    <motion.button
      ref={setNodeRef}
      type="button"
      style={style}
      className={cn(
        "relative grid shrink-0 place-items-center border-0 bg-transparent p-0 touch-pan-x cursor-grab active:cursor-grabbing xl:touch-none",
        gameCardFrameSizeClassNames[size],
      )}
      initial="rest"
      animate="rest"
      whileHover="hover"
      whileTap="pressed"
      title={accessibleName}
      aria-label={accessibleName}
      {...listeners}
      {...attributes}
    >
      <motion.span
        className={cn(
          faceDown
            ? faceDownGameCardClassName(size, className)
            : gameCardClassName(card, size, className),
          decorationRingClassName(decoration?.highlight),
          invalidRingClassName(invalid),
          "transition-none",
        )}
        custom={shouldReduceMotion}
        variants={cardMotionVariants}
      >
        {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
        <GameCardDecorationLayer decoration={decoration} />
      </motion.span>
    </motion.button>
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
  invalid,
}: {
  card: CardSnapshot;
  id: string;
  size: GameCardSize;
  className?: string;
  disabled?: boolean;
  faceDown?: boolean;
  data?: Record<string, unknown>;
  decoration?: GameCardDecoration;
  invalid?: boolean;
}) {
  const shouldReduceMotion = useShouldReduceMotion();
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id,
    disabled,
    data,
  });
  const accessibleName = faceDown ? m.face_down_card() : cardName(card);

  const style: CSSProperties = {
    transform: transform ? `translate3d(${transform.x}px, ${transform.y}px, 0)` : undefined,
    opacity: isDragging ? 0 : 1,
    zIndex: isDragging ? 20 : undefined,
  };

  return (
    <motion.button
      ref={setNodeRef}
      type="button"
      style={style}
      disabled={disabled}
      className={cn(
        "relative grid shrink-0 place-items-center border-0 bg-transparent p-0",
        gameCardFrameSizeClassNames[size],
        disabled
          ? "cursor-default opacity-50"
          : "touch-pan-x cursor-grab active:cursor-grabbing xl:touch-none",
      )}
      initial="rest"
      animate="rest"
      whileHover={disabled ? undefined : "hover"}
      whileTap={disabled ? undefined : "pressed"}
      title={accessibleName}
      aria-label={accessibleName}
      {...listeners}
      {...attributes}
    >
      <motion.span
        className={cn(
          faceDown
            ? faceDownGameCardClassName(size, className)
            : gameCardClassName(card, size, className),
          decorationRingClassName(decoration?.highlight),
          invalidRingClassName(invalid),
          "transition-none",
        )}
        custom={shouldReduceMotion}
        variants={cardMotionVariants}
      >
        {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
        <GameCardDecorationLayer decoration={decoration} />
      </motion.span>
    </motion.button>
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
  invalid,
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
  invalid?: boolean;
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
        data={{ card, isVirtual: draggable.isVirtual }}
        decoration={decoration}
        invalid={invalid}
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
        invalid={invalid}
      />
    );
  }

  const accessibleName = faceDown ? m.face_down_card() : cardName(card);

  return (
    <div
      className={
        faceDown
          ? faceDownGameCardClassName(
              size,
              cn(
                className,
                decorationRingClassName(decoration?.highlight),
                invalidRingClassName(invalid),
              ),
            )
          : gameCardClassName(
              card,
              size,
              cn(
                className,
                decorationRingClassName(decoration?.highlight),
                invalidRingClassName(invalid),
              ),
            )
      }
      title={accessibleName}
      aria-label={accessibleName}
      aria-invalid={invalid || undefined}
    >
      {faceDown ? renderGameCardBack() : renderGameCardFace(card, size)}
      <GameCardDecorationLayer decoration={decoration} />
    </div>
  );
}

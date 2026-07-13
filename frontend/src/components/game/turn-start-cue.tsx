import { useEffect, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { getUserInitials } from "#/lib/utils";

export function TurnStartCue({
  round,
  turnNumber,
  playerName,
  playerImageUrl,
}: {
  round: number;
  turnNumber: number;
  playerName: string;
  playerImageUrl?: string;
}) {
  const [visible, setVisible] = useState(true);
  const shouldReduceMotion = useReducedMotion();
  const turnKey = `${round}:${turnNumber}`;

  useEffect(() => {
    setVisible(true);
    const hideTimer = window.setTimeout(() => setVisible(false), 1_600);

    return () => window.clearTimeout(hideTimer);
  }, [turnKey]);

  return (
    <AnimatePresence>
      {visible ? (
        <motion.div
          key={turnKey}
          className="pointer-events-none fixed top-[clamp(4.5rem,12vh,8rem)] left-1/2 z-50 w-[min(22rem,calc(100vw-2rem))] will-change-[clip-path,transform,opacity]"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          aria-label={`Your turn, ${playerName}`}
          data-turn-number={turnNumber}
          initial={{
            opacity: 0,
            clipPath: shouldReduceMotion ? "inset(0 0 0 0)" : "inset(0 100% 0 0)",
            transform: shouldReduceMotion
              ? "translateX(-50%)"
              : "translateX(calc(-50% - 10px)) rotate(-0.8deg)",
          }}
          animate={{
            opacity: 1,
            clipPath: "inset(0 0 0 0)",
            transform: "translateX(-50%) rotate(0deg)",
          }}
          exit={{
            opacity: 0,
            clipPath: shouldReduceMotion ? "inset(0 0 0 0)" : "inset(0 0 0 100%)",
            transform: shouldReduceMotion
              ? "translateX(-50%)"
              : "translateX(calc(-50% + 8px)) rotate(0.5deg)",
          }}
          transition={{
            duration: shouldReduceMotion ? 0.15 : 0.28,
            ease: [0.23, 1, 0.32, 1],
          }}
        >
          <div className="relative flex h-[4.5rem] items-stretch bg-foreground text-background shadow-[5px_5px_0_color-mix(in_oklab,var(--primary)_72%,var(--foreground))] [clip-path:polygon(0_0,calc(100%-12px)_0,100%_12px,100%_100%,12px_100%,0_calc(100%-12px))]">
            <div className="relative flex w-[4.5rem] shrink-0 items-center justify-center border-r border-background/20">
              <Avatar className="relative size-10 border border-background/35 bg-foreground">
                {playerImageUrl ? (
                  <AvatarImage src={playerImageUrl} alt={`${playerName}'s avatar`} />
                ) : null}
                <AvatarFallback className="bg-background text-xs font-bold tracking-tight text-foreground">
                  {getUserInitials(playerName)}
                </AvatarFallback>
                <AvatarBadge className="right-0 bottom-0 size-2.5 border border-foreground bg-primary ring-0" />
              </Avatar>
            </div>

            <div className="flex min-w-0 flex-1 items-center justify-between gap-4 px-4">
              <div className="min-w-0">
                <p className="mb-1 flex items-center gap-2 text-[0.55rem] leading-none font-medium tracking-[0.2em] text-background/55 uppercase">
                  <span className="inline-block size-1.5 rotate-45 bg-primary" aria-hidden="true" />
                  Table is yours
                </p>
                <p className="truncate text-[1.05rem] leading-none font-bold tracking-[-0.04em] uppercase">
                  Your turn
                </p>
              </div>
              <p className="shrink-0 text-right text-[0.5rem] leading-[1.45] tracking-[0.16em] text-background/45 uppercase tabular-nums">
                R{String(round).padStart(2, "0")}
                <br />T{String(turnNumber).padStart(2, "0")}
              </p>
            </div>

            <motion.span
              aria-hidden="true"
              className="absolute inset-y-0 left-0 w-px bg-primary"
              initial={shouldReduceMotion ? false : { opacity: 0, transform: "translateX(0)" }}
              animate={
                shouldReduceMotion
                  ? { opacity: 0 }
                  : { opacity: [0, 1, 1, 0], transform: "translateX(22rem)" }
              }
              transition={{ duration: 0.5, ease: [0.23, 1, 0.32, 1] }}
            />
          </div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

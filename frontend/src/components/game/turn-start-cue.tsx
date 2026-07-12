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
          className="pointer-events-none fixed top-1/2 left-1/2 z-50 flex w-[min(22rem,calc(100vw-2rem))] items-center gap-4 overflow-hidden rounded-[2rem] border border-primary/30 bg-popover/95 p-3 pr-7 text-foreground shadow-[0_1.5rem_5rem_-1.5rem_color-mix(in_oklab,var(--primary)_35%,transparent),0_0.75rem_2rem_-1rem_rgb(0_0_0/0.4)] ring-1 ring-foreground/8 backdrop-blur-2xl backdrop-saturate-150 will-change-transform before:absolute before:inset-y-0 before:left-0 before:w-28 before:bg-gradient-to-r before:from-primary/15 before:to-transparent"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          aria-label={`Your turn, ${playerName}`}
          data-turn-number={turnNumber}
          initial={{
            opacity: 0,
            transform: shouldReduceMotion
              ? "translate(-50%, -50%)"
              : "translate(-50%, -46%) scale(0.96)",
          }}
          animate={{ opacity: 1, transform: "translate(-50%, -50%) scale(1)" }}
          exit={{
            opacity: 0,
            transform: shouldReduceMotion
              ? "translate(-50%, -50%)"
              : "translate(-50%, -54%) scale(0.98)",
          }}
          transition={{ duration: shouldReduceMotion ? 0.15 : 0.2, ease: [0.23, 1, 0.32, 1] }}
        >
          <Avatar className="relative size-16 border-2 border-background shadow-lg ring-2 ring-primary/40">
            {playerImageUrl ? (
              <AvatarImage src={playerImageUrl} alt={`${playerName}'s avatar`} />
            ) : null}
            <AvatarFallback className="bg-primary/12 text-lg font-semibold text-primary">
              {getUserInitials(playerName)}
            </AvatarFallback>
            <AvatarBadge className="right-0.5 bottom-0.5 size-4 bg-primary ring-4 ring-popover" />
          </Avatar>
          <div className="relative min-w-0">
            <p className="mb-0.5 text-[0.625rem] font-semibold tracking-[0.22em] text-primary uppercase">
              Make your move
            </p>
            <p className="text-2xl font-semibold tracking-tight text-balance">Your turn</p>
          </div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

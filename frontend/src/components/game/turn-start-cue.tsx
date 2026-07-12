import { useEffect, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";

export function TurnStartCue({ round, turnNumber }: { round: number; turnNumber: number }) {
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
          className="pointer-events-none fixed top-[max(1rem,env(safe-area-inset-top))] left-1/2 z-50 inline-flex items-center gap-2 rounded-full border border-primary/30 bg-popover/90 px-3.5 py-2 text-xs font-semibold tracking-[0.02em] text-foreground shadow-lg ring-1 ring-foreground/5 backdrop-blur-xl backdrop-saturate-150 will-change-transform"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          data-turn-number={turnNumber}
          initial={{
            opacity: 0,
            transform: shouldReduceMotion
              ? "translateX(-50%)"
              : "translate(-50%, -0.5rem) scale(0.98)",
          }}
          animate={{ opacity: 1, transform: "translate(-50%, 0) scale(1)" }}
          exit={{
            opacity: 0,
            transform: shouldReduceMotion
              ? "translateX(-50%)"
              : "translate(-50%, -0.25rem) scale(0.99)",
          }}
          transition={{ duration: shouldReduceMotion ? 0.15 : 0.2, ease: [0.23, 1, 0.32, 1] }}
        >
          <motion.span
            className="size-1.5 rounded-full bg-primary shadow-[0_0_0.625rem] shadow-primary/50"
            aria-hidden="true"
            animate={
              shouldReduceMotion
                ? undefined
                : {
                    opacity: [0.55, 1, 0.55],
                    transform: ["scale(0.8)", "scale(1)", "scale(0.8)"],
                  }
            }
            transition={{ duration: 0.7, delay: 0.1, times: [0, 0.45, 1] }}
          />
          <span>Your turn</span>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

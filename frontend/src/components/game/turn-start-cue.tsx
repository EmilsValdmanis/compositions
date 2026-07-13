import { useEffect, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Separator } from "#/components/ui/separator";
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
          className="pointer-events-none fixed top-[clamp(4.5rem,12vh,8rem)] left-1/2 z-50 w-[min(22rem,calc(100vw-2rem))] will-change-[transform,opacity]"
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
          <Alert
            className="h-18 grid-cols-[4.5rem_auto_minmax(0,1fr)] grid-rows-1 gap-0 overflow-hidden rounded-3xl p-0 shadow-lg"
            role="status"
            aria-live="polite"
            aria-atomic="true"
            aria-label={`Your turn, ${playerName}`}
            data-turn-number={turnNumber}
          >
            <div className="grid place-items-center bg-muted/30">
              <Avatar className="size-10">
                {playerImageUrl ? (
                  <AvatarImage src={playerImageUrl} alt={`${playerName}'s avatar`} />
                ) : null}
                <AvatarFallback>{getUserInitials(playerName)}</AvatarFallback>
                <AvatarBadge />
              </Avatar>
            </div>
            <Separator orientation="vertical" />

            <div className="flex min-w-0 items-center justify-between gap-3 px-4">
              <div className="min-w-0">
                <AlertDescription className="mb-1 flex items-center gap-2 text-[0.55rem] leading-none font-medium tracking-[0.2em] uppercase">
                  <span className="size-1.5 rotate-45 bg-primary" aria-hidden="true" />
                  Table is yours
                </AlertDescription>
                <AlertTitle className="truncate text-[1.05rem] leading-none font-bold tracking-[-0.04em] uppercase">
                  Your turn
                </AlertTitle>
              </div>
              <p className="shrink-0 text-right text-[0.5rem] leading-[1.45] tracking-[0.16em] text-muted-foreground uppercase tabular-nums">
                R{String(round).padStart(2, "0")}
                <br />T{String(turnNumber).padStart(2, "0")}
              </p>
            </div>
          </Alert>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

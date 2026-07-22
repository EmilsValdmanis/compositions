import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Alert } from "#/components/ui/alert";
import { Avatar, AvatarBadge, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Separator } from "#/components/ui/separator";
import { Caption, H4 } from "#/components/typography";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { getUserInitials } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

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
  const cueRef = useRef<HTMLDivElement>(null);
  const shouldReduceMotion = useShouldReduceMotion();

  useEffect(() => {
    const hideTimer = window.setTimeout(() => setVisible(false), 1_600);

    return () => window.clearTimeout(hideTimer);
  }, []);

  return (
    <AnimatePresence>
      {visible ? (
        <motion.div
          data-slot="turn-start-cue"
          className="pointer-events-none absolute inset-0 z-50 grid place-items-center p-4"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: shouldReduceMotion ? 0.15 : 0.2 }}
        >
          <motion.div
            ref={cueRef}
            className="w-[min(22rem,calc(100vw-2rem))]"
            onAnimationStart={() => {
              if (cueRef.current) cueRef.current.style.willChange = "transform";
            }}
            onAnimationComplete={() => {
              if (cueRef.current) cueRef.current.style.willChange = "";
            }}
            initial={{ scale: shouldReduceMotion ? 1 : 0.96 }}
            animate={{ scale: 1 }}
            transition={{
              duration: shouldReduceMotion ? 0.15 : 0.28,
              ease: [0.23, 1, 0.32, 1],
            }}
          >
            <Alert
              className="h-18 grid-cols-[4.5rem_auto_minmax(0,1fr)] grid-rows-1 gap-0 overflow-hidden p-0 shadow-lg"
              role="status"
              aria-live="polite"
              aria-atomic="true"
              aria-label={m.your_turn_named({ name: playerName })}
              data-turn-number={turnNumber}
            >
              <div className="grid place-items-center bg-muted/30">
                <Avatar className="size-10">
                  {playerImageUrl ? (
                    <AvatarImage src={playerImageUrl} alt={m.player_avatar({ name: playerName })} />
                  ) : null}
                  <AvatarFallback>{getUserInitials(playerName)}</AvatarFallback>
                  <AvatarBadge />
                </Avatar>
              </div>
              <Separator orientation="vertical" />

              <div className="flex min-w-0 items-center justify-between gap-3 px-4">
                <div className="min-w-0">
                  <Caption
                    data-slot="alert-description"
                    className="mb-1 flex items-center gap-2 text-[0.5rem]/none font-medium tracking-[0.2em] uppercase md:text-[0.55rem]/none"
                  >
                    <span className="size-1.5 rotate-45 bg-primary" aria-hidden="true" />
                    {m.table_is_yours()}
                  </Caption>
                  <H4
                    className="truncate text-[0.95rem]/none font-bold tracking-[-0.04em] uppercase md:text-[1.05rem]/none"
                    data-slot="alert-title"
                  >
                    {m.your_turn()}
                  </H4>
                </div>
                <Caption className="shrink-0 text-right text-[0.45rem]/[1.45] tracking-[0.16em] uppercase tabular-nums md:text-[0.5rem]/[1.45]">
                  R{String(round).padStart(2, "0")}
                  <br />T{String(turnNumber).padStart(2, "0")}
                </Caption>
              </div>
            </Alert>
          </motion.div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}

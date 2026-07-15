import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { motion } from "motion/react";
import { Popover, PopoverContent, PopoverTrigger } from "#/components/ui/popover";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { useShouldReduceMotion } from "#/lib/reduced-motion";
import { m } from "#/paraglide/messages.js";

export const PLAYER_EMOTES = [
  "👋",
  "👍",
  "😂",
  "😅",
  "🤔",
  "😮",
  "😡",
  "👀",
  "😭",
  "🔥",
  "❤️",
  "🎉",
] as const;

const emojiFont = {
  fontFamily: '"Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji", sans-serif',
};

export function PlayerEmoteBubble({
  emote,
}: {
  emote: { id: string; emoji: string; expiresAt: string };
}) {
  const shouldReduceMotion = useShouldReduceMotion();
  const anchorRef = useRef<HTMLSpanElement>(null);
  const [position, setPosition] = useState<{ left: number; top: number } | null>(null);

  useLayoutEffect(() => {
    const anchor = anchorRef.current;
    if (!anchor) return;

    function updatePosition() {
      const currentAnchor = anchorRef.current;
      if (!currentAnchor) return;

      const rect = currentAnchor.getBoundingClientRect();
      setPosition((current) =>
        current?.left === rect.left && current.top === rect.top
          ? current
          : { left: rect.left, top: rect.top },
      );
    }

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);

    const resizeObserver =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(updatePosition);
    resizeObserver?.observe(anchor.parentElement ?? anchor);

    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
      resizeObserver?.disconnect();
    };
  }, [emote.id]);

  return (
    <>
      <span ref={anchorRef} className="pointer-events-none absolute -top-6 left-8 size-0" />
      {position
        ? createPortal(
            <Badge
              key={emote.id}
              variant="outline"
              aria-label={m.player_emote()}
              render={
                <motion.span
                  initial={{
                    opacity: 0,
                    y: shouldReduceMotion ? 0 : 4,
                    scale: shouldReduceMotion ? 1 : 0.78,
                  }}
                  animate={{
                    opacity: [0, 1, 1, 0],
                    y: shouldReduceMotion ? 0 : [4, -4, -2, -14],
                    scale: shouldReduceMotion ? 1 : [0.78, 1.08, 1, 0.9],
                  }}
                  transition={{
                    duration: 4,
                    ease: "easeOut",
                    times: [0, 0.12, 0.72, 1],
                  }}
                />
              }
              className="pointer-events-none fixed z-50 grid size-9 place-items-center rounded-full bg-background/95 p-0 shadow-lg ring-1 ring-foreground/5"
              style={{ ...emojiFont, left: position.left, top: position.top }}
            >
              <span className="text-xl/none">{emote.emoji}</span>
            </Badge>,
            document.body,
          )
        : null}
    </>
  );
}

export function PlayerEmotePicker({
  onSendEmote,
  className,
}: {
  onSendEmote: (emoji: string) => void;
  className?: string;
}) {
  const [open, setOpen] = useState(false);

  function handleSendEmote(emoji: string) {
    onSendEmote(emoji);
    setOpen(false);
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            className={className}
            aria-label={m.open_emotes()}
          />
        }
      >
        <span className="text-base/none" aria-hidden style={emojiFont}>
          🙂
        </span>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={8} className="w-auto rounded-2xl p-2">
        <div className="grid grid-cols-4 gap-1">
          {PLAYER_EMOTES.map((emoji) => (
            <Button
              key={emoji}
              type="button"
              variant="ghost"
              size="icon-sm"
              className="hover:scale-105"
              aria-label={m.send_emote({ emoji })}
              onClick={() => handleSendEmote(emoji)}
            >
              <span className="text-lg/none" aria-hidden style={emojiFont}>
                {emoji}
              </span>
            </Button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

import { useState } from "react";
import { motion } from "motion/react";
import { Popover, PopoverContent, PopoverTrigger } from "#/components/ui/popover";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Text } from "#/components/typography";
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

  return (
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
      className="pointer-events-none absolute -top-6 left-8 z-20 grid size-9 place-items-center rounded-full bg-background/95 p-0 shadow-lg ring-1 ring-foreground/5"
      style={emojiFont}
    >
      <Text variant="emoji-xl">{emote.emoji}</Text>
    </Badge>
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
        <Text variant="emoji-base" aria-hidden style={emojiFont}>
          🙂
        </Text>
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
              <Text variant="emoji-lg" aria-hidden style={emojiFont}>
                {emoji}
              </Text>
            </Button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

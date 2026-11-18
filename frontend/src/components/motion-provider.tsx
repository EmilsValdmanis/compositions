import { MotionConfig } from "motion/react";
import { ScriptOnce } from "@tanstack/react-router";
import { REDUCED_MOTION_SCRIPT, useShouldReduceMotion } from "#/lib/reduced-motion";

export function MotionProvider({ children }: { children: React.ReactNode }) {
  const shouldReduceMotion = useShouldReduceMotion();

  return (
    <MotionConfig reducedMotion={shouldReduceMotion ? "always" : "never"}>
      <ScriptOnce>{REDUCED_MOTION_SCRIPT}</ScriptOnce>
      {children}
    </MotionConfig>
  );
}

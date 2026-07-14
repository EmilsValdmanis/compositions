import NumberFlow, { type NumberFlowProps } from "@number-flow/react";
import { cn } from "#/lib/utils";
import { useShouldReduceMotion } from "#/lib/reduced-motion";

const numberTiming: KeyframeAnimationOptions = {
  duration: 260,
  easing: "cubic-bezier(0.23, 1, 0.32, 1)",
  fill: "both",
};

const opacityTiming: KeyframeAnimationOptions = {
  duration: 180,
  easing: "cubic-bezier(0.23, 1, 0.32, 1)",
  fill: "both",
};

type AnimatedNumberProps = Omit<
  NumberFlowProps,
  "transformTiming" | "spinTiming" | "opacityTiming" | "respectMotionPreference"
>;

/** Shared motion treatment for live game values. */
export function AnimatedNumber({ className, ...props }: AnimatedNumberProps) {
  const shouldReduceMotion = useShouldReduceMotion();

  return (
    <NumberFlow
      className={cn("inline-block tabular-nums", className)}
      transformTiming={numberTiming}
      spinTiming={numberTiming}
      opacityTiming={opacityTiming}
      animated={!shouldReduceMotion}
      respectMotionPreference
      isolate
      {...props}
    />
  );
}

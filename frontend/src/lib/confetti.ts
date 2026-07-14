import type confetti from "canvas-confetti";

type CelebrationConfettiOptions = {
  count?: number;
  originY?: number;
  delayMs?: number;
};

const APP_COLORS = ["#a5d8f3", "#55b6e8", "#278bc7", "#176ca5", "#0d4d7d"];

export async function fireCelebrationConfetti({
  count = 200,
  originY = 0.7,
  delayMs = 0,
}: CelebrationConfettiOptions = {}) {
  if (
    typeof window === "undefined" ||
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  ) {
    return;
  }

  const confettiModule = await import("canvas-confetti");
  const fireConfetti = confettiModule.default;

  if (delayMs > 0) {
    await new Promise((resolve) => window.setTimeout(resolve, delayMs));
  }

  const defaults: confetti.Options = {
    colors: APP_COLORS,
    origin: { y: originY },
  };

  const fire = (particleRatio: number, options: confetti.Options) => {
    void fireConfetti({
      ...defaults,
      ...options,
      particleCount: Math.floor(count * particleRatio),
    });
  };

  fire(0.25, { spread: 26, startVelocity: 55 });
  fire(0.2, { spread: 60 });
  fire(0.35, { spread: 100, decay: 0.91, scalar: 0.8 });
  fire(0.1, { spread: 120, startVelocity: 25, decay: 0.92, scalar: 1.2 });
  fire(0.1, { spread: 120, startVelocity: 45 });
}

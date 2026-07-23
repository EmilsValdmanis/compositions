export const SPECTATOR_MOTION_EASE_OUT = [0.23, 1, 0.32, 1] as const;
export const SPECTATOR_MOTION_EASE_IN_OUT = [0.77, 0, 0.175, 1] as const;

export const spectatorCardEnter = {
  opacity: 0,
  scale: 0.96,
  y: 6,
};

export const spectatorCardVisible = {
  opacity: 1,
  scale: 1,
  y: 0,
};

export const spectatorCardExit = {
  opacity: 0,
  scale: 0.96,
  y: -4,
};

export const spectatorCardTransition = {
  duration: 0.2,
  ease: SPECTATOR_MOTION_EASE_OUT,
  layout: {
    duration: 0.24,
    ease: SPECTATOR_MOTION_EASE_IN_OUT,
  },
};

export const spectatorCardExitTransition = {
  duration: 0.14,
  ease: SPECTATOR_MOTION_EASE_OUT,
};

export const spectatorCompositionEnter = {
  opacity: 0,
  scale: 0.985,
  y: 4,
};

export const spectatorCompositionExit = {
  opacity: 0,
  scale: 0.985,
  y: -3,
};

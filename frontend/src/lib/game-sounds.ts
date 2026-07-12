import { tiks } from "@rexa-developer/tiks";

const SOUND_PREFERENCE_KEY = "compositions.game-sounds-enabled";
const preferenceListeners = new Set<() => void>();

export type GameSound =
  | "card-pickup"
  | "card-place"
  | "card-draw"
  | "card-discard"
  | "composition-create"
  | "joker-reclaim"
  | "turn-start"
  | "invalid-action"
  | "player-joined"
  | "player-left"
  | "round-win"
  | "game-win";

export function areGameSoundsEnabled() {
  if (typeof window === "undefined") {
    return true;
  }

  return window.localStorage.getItem(SOUND_PREFERENCE_KEY) !== "false";
}

export function subscribeToGameSoundPreference(listener: () => void) {
  preferenceListeners.add(listener);
  return () => preferenceListeners.delete(listener);
}

tiks.init({
  theme: "soft",
  volume: 0.14,
  muted: !areGameSoundsEnabled(),
});

const soundPlayers: Record<GameSound, () => void> = {
  "card-pickup": () => tiks.click(),
  "card-place": () => tiks.pop(),
  "card-draw": () => tiks.swoosh(),
  "card-discard": () => tiks.toggle(false),
  "composition-create": () => tiks.success(),
  "joker-reclaim": () => tiks.toggle(true),
  "turn-start": () => tiks.notify(),
  "invalid-action": () => tiks.error(),
  "player-joined": () => tiks.pop(),
  "player-left": () => tiks.toggle(false),
  "round-win": () => tiks.success(),
  "game-win": () => tiks.notify(),
};

export function playGameSound(sound: GameSound) {
  soundPlayers[sound]();
}

export function setGameSoundsEnabled(enabled: boolean) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(SOUND_PREFERENCE_KEY, String(enabled));
  }

  if (enabled) {
    tiks.unmute();
    tiks.toggle(true);
  } else {
    tiks.toggle(false);
    tiks.mute();
  }

  for (const listener of preferenceListeners) {
    listener();
  }
}

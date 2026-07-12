import { useEffect, useRef } from "react";
import { type LobbyState } from "#/components/game-websocket-provider";
import { type GameSound, playGameSound } from "#/lib/game-sounds";

function playerIds(state: LobbyState) {
  return new Set(state.room?.players.map((player) => player.playerId) ?? []);
}

function compositionActivitySignature(state: LobbyState) {
  return JSON.stringify(state.game?.turnActivity?.compositionActivities ?? []);
}

function hasJokerReclaimActivity(state: LobbyState) {
  return (state.game?.turnActivity?.compositionActivities ?? []).some(
    (composition) =>
      composition.kind === "joker_reclaim" ||
      Object.values(composition.cardActivities ?? {}).some(
        (activity) => activity.kind === "joker_reclaim",
      ),
  );
}

export function gameSoundsForStateChange(previous: LobbyState, current: LobbyState): GameSound[] {
  const sounds: GameSound[] = [];

  if (current.lastError && current.lastErrorId !== previous.lastErrorId) {
    sounds.push("invalid-action");
  }

  const isSameRoom = Boolean(
    previous.room?.code && current.room?.code && previous.room.code === current.room.code,
  );

  if (!isSameRoom) {
    return sounds;
  }

  const previousPlayerIds = playerIds(previous);
  const currentPlayerIds = playerIds(current);

  if ([...currentPlayerIds].some((playerId) => !previousPlayerIds.has(playerId))) {
    sounds.push("player-joined");
  }
  if ([...previousPlayerIds].some((playerId) => !currentPlayerIds.has(playerId))) {
    sounds.push("player-left");
  }

  if (current.room?.phase === "game_over" && previous.room?.phase !== "game_over") {
    sounds.push("game-win");
  } else if (current.room?.phase === "round_over" && previous.room?.phase !== "round_over") {
    sounds.push("round-win");
  }

  if (!previous.game || !current.game || previous.game.round !== current.game.round) {
    if (current.room?.phase === "in_progress" && current.game?.turn.playerId === current.playerId) {
      sounds.push("turn-start");
    }
    return sounds;
  }

  if (
    previous.game.turn.number === current.game.turn.number &&
    !previous.game.turn.hasDrawn &&
    current.game.turn.hasDrawn
  ) {
    sounds.push("card-draw");
  }

  if (previous.game.turn.number !== current.game.turn.number) {
    sounds.push("card-discard");
    if (current.room?.phase === "in_progress" && current.game.turn.playerId === current.playerId) {
      sounds.push("turn-start");
    }
  }

  if (
    (current.game.turnActivity?.compositionActivities?.length ?? 0) > 0 &&
    compositionActivitySignature(previous) !== compositionActivitySignature(current)
  ) {
    const createdComposition =
      current.game.activeCompositions.length > previous.game.activeCompositions.length;
    const reclaimedJoker = hasJokerReclaimActivity(current);

    if (createdComposition) {
      sounds.push("composition-create");
    }
    if (reclaimedJoker) {
      sounds.push("joker-reclaim");
    }
    if (!createdComposition && !reclaimedJoker) {
      sounds.push("card-place");
    }
  }

  return sounds;
}

export function useGameSoundEvents(state: LobbyState) {
  const previousStateRef = useRef(state);

  useEffect(() => {
    const previousState = previousStateRef.current;
    previousStateRef.current = state;

    for (const sound of gameSoundsForStateChange(previousState, state)) {
      playGameSound(sound);
    }
  }, [state]);
}

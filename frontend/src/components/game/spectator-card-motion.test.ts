import { describe, expect, it } from "vite-plus/test";
import {
  spectatorCardExitTransition,
  spectatorCardTransition,
} from "#/components/game/spectator-card-motion";

describe("spectator card motion", () => {
  it("keeps card feedback quick and removals faster than placement", () => {
    expect(spectatorCardTransition.duration).toBeLessThan(0.3);
    expect(spectatorCardTransition.layout.duration).toBeLessThan(0.3);
    expect(spectatorCardExitTransition.duration).toBeLessThan(spectatorCardTransition.duration);
  });
});

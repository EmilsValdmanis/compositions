import { describe, expect, it } from "vite-plus/test";
import { mockScenarios } from "#/dev/mock-game-scenarios";

describe("dev game scenarios", () => {
  it("showcases the add and joker-reclaim card labels on the board", () => {
    const showcase = mockScenarios.find((scenario) => scenario.id === "table-activity-showcase");
    const activities = showcase?.game.turnActivity?.compositionActivities ?? [];

    expect(
      activities.some((activity) =>
        Object.values(activity.cardActivities ?? {}).some(
          (cardActivity) => cardActivity.kind === "addition",
        ),
      ),
    ).toBe(true);
    expect(
      activities.some((activity) =>
        Object.values(activity.cardActivities ?? {}).some(
          (cardActivity) => cardActivity.kind === "joker_reclaim",
        ),
      ),
    ).toBe(true);
  });
});

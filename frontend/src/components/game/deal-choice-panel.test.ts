import { describe, expect, it } from "vite-plus/test";
import { deckVisualizationLayerCounts } from "#/components/game/deal-choice-panel";

describe("deckVisualizationLayerCounts", () => {
  it("preserves cut proportions across a fixed set of transferable cards", () => {
    expect(deckVisualizationLayerCounts(0)).toEqual({ lifted: 0, remaining: 18 });
    expect(deckVisualizationLayerCounts(12)).toEqual({ lifted: 2, remaining: 16 });
    expect(deckVisualizationLayerCounts(30)).toEqual({ lifted: 5, remaining: 13 });
    expect(deckVisualizationLayerCounts(60)).toEqual({ lifted: 10, remaining: 8 });
    expect(deckVisualizationLayerCounts(108)).toEqual({ lifted: 18, remaining: 0 });
  });
});

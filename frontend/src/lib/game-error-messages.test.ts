import { describe, expect, it } from "vite-plus/test";
import { gameErrorMessage } from "#/lib/game-error-messages";
import { m } from "#/paraglide/messages.js";

describe("gameErrorMessage", () => {
  it("does not resolve inherited object property names", () => {
    expect(gameErrorMessage("toString")).toBe(m.error_unknown());
    expect(gameErrorMessage("__proto__")).toBe(m.error_unknown());
  });
});

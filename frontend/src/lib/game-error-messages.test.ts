import { describe, expect, it } from "vite-plus/test";
import { gameErrorMessage } from "#/lib/game-error-messages";
import { m } from "#/paraglide/messages.js";

describe("gameErrorMessage", () => {
  it("explains that an unconfirmed command may already have completed", () => {
    expect(gameErrorMessage("command_timeout")).toBe(m.error_command_timeout());
    expect(gameErrorMessage("command_timeout")).not.toBe(m.error_unknown());
  });
  it("does not resolve inherited object property names", () => {
    expect(gameErrorMessage("toString")).toBe(m.error_unknown());
    expect(gameErrorMessage("__proto__")).toBe(m.error_unknown());
  });
});

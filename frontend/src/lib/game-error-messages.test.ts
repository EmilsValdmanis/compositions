import { afterEach, describe, expect, it } from "vite-plus/test";
import { gameErrorMessage } from "#/lib/game-error-messages";
import { m } from "#/paraglide/messages.js";
import { getLocale, overwriteGetLocale } from "#/paraglide/runtime.js";

const originalGetLocale = getLocale;

afterEach(() => {
  overwriteGetLocale(originalGetLocale);
});

describe("gameErrorMessage", () => {
  it("explains that an unconfirmed command may already have completed", () => {
    expect(gameErrorMessage("command_timeout")).toBe(m.error_command_timeout());
    expect(gameErrorMessage("command_timeout")).not.toBe(m.error_unknown());
  });
  it("does not resolve inherited object property names", () => {
    expect(gameErrorMessage("toString")).toBe(m.error_unknown());
    expect(gameErrorMessage("__proto__")).toBe(m.error_unknown());
  });

  it("uses the active locale for known codes and the unknown-code fallback", () => {
    overwriteGetLocale(() => "en");
    const english = gameErrorMessage("room_full");

    overwriteGetLocale(() => "lv");
    expect(gameErrorMessage("room_full")).toBe(m.error_room_full());
    expect(gameErrorMessage("room_full")).not.toBe(english);
    expect(gameErrorMessage(null)).toBe(m.error_unknown());
  });
});

import { describe, expect, it } from "vitest";

import { resolveGameConnectionAuth } from "#/lib/game-connection-auth";

describe("resolveGameConnectionAuth", () => {
  it("returns the signed session token and user details", () => {
    const headers = new Headers({
      cookie:
        "compositions.session_token=signed.token; compositions.session_data=encrypted.session; theme=dark",
    });

    const auth = resolveGameConnectionAuth(headers, {
      user: {
        id: "user-1",
        name: "Player One",
        email: "player@example.com",
      },
    });

    expect(auth).toEqual({
      authToken: "signed.token",
      user: {
        id: "user-1",
        name: "Player One",
        email: "player@example.com",
      },
    });
  });

  it("accepts secure cookie names", () => {
    const headers = new Headers({
      cookie:
        "__Secure-compositions.session_token=signed.secure.token; __Secure-compositions.session_data.0=part-one; __Secure-compositions.session_data.1=part-two",
    });

    const auth = resolveGameConnectionAuth(headers, {
      user: {
        id: "user-1",
        name: "Player One",
        email: "player@example.com",
      },
    });

    expect(auth?.authToken).toBe("signed.secure.token");
    expect(auth?.user.id).toBe("user-1");
  });

  it("returns null when the session is missing", () => {
    const headers = new Headers({
      cookie: "compositions.session_token=signed.token",
    });

    expect(resolveGameConnectionAuth(headers, null)).toBeNull();
  });

  it("returns null when the session cookie is missing", () => {
    const headers = new Headers({
      cookie: "theme=dark",
    });

    expect(
      resolveGameConnectionAuth(headers, {
        user: {
          id: "user-1",
          name: "Player One",
          email: "player@example.com",
        },
      }),
    ).toBeNull();
  });
});

import { describe, expect, it, vi } from "vitest";

import { loadVerifiedSession } from "#/lib/verified-session";

describe("loadVerifiedSession", () => {
  it("disables cookie cache for verified session reads", async () => {
    const headers = new Headers({ cookie: "compositions.session_token=signed.token" });
    const session = { user: { id: "user-1" } };
    const getSession = vi.fn().mockResolvedValue(session);

    const result = await loadVerifiedSession(headers, getSession);

    expect(result).toBe(session);
    expect(getSession).toHaveBeenCalledWith({
      headers,
      query: {
        disableCookieCache: true,
      },
    });
  });

  it("returns null when the verified session lookup fails", async () => {
    const headers = new Headers({ cookie: "compositions.session_token=signed.token" });
    const getSession = vi.fn().mockResolvedValue(null);

    await expect(loadVerifiedSession(headers, getSession)).resolves.toBeNull();
  });
});

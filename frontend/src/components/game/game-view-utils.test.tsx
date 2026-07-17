// @vitest-environment jsdom

import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { AddActivityLabel, ReclaimActivityLabel } from "#/components/game/game-view-utils";
import { m } from "#/paraglide/messages.js";

afterEach(cleanup);

describe("compact activity labels", () => {
  it.each([
    [AddActivityLabel, m.activity_add()],
    [ReclaimActivityLabel, m.activity_reclaim()],
  ] as const)("renders the translated label accessibly behind an icon", (Label, title) => {
    const view = render(<Label players={[]} />);
    const label = view.getByTitle(title);

    expect(label.querySelector("svg")).not.toBeNull();
    expect(label.querySelector(".sr-only")?.textContent).toBe(title);
  });
});

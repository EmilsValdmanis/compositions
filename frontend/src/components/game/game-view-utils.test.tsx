// @vitest-environment jsdom

import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";
import { AddActivityLabel, ReclaimActivityLabel } from "#/components/game/game-view-utils";
import { m } from "#/paraglide/messages.js";

afterEach(cleanup);

describe("activity labels", () => {
  it.each([
    [AddActivityLabel, m.activity_add()],
    [ReclaimActivityLabel, m.activity_reclaim()],
  ] as const)("renders both the translated label and its icon", (Label, title) => {
    const view = render(<Label players={[]} />);
    const label = view.getByText(title);

    expect(label.classList.contains("sr-only")).toBe(false);
    expect(label.parentElement?.querySelector("svg")).not.toBeNull();
  });
});

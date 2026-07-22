// @vitest-environment jsdom
import { createElement, type ReactNode } from "react";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import { SidebarFriendsList, formatGameDuration } from "#/components/social/friends-list";
import { SidebarProvider } from "#/components/ui/sidebar";

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => vi.fn(async () => undefined),
  Link: ({
    children,
    to: _to,
    params: _params,
    ...props
  }: {
    children: ReactNode;
    to?: unknown;
    params?: unknown;
    [key: string]: unknown;
  }) => createElement("a", { href: "/", ...props }, children),
}));

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation(() => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })),
});

afterEach(cleanup);

describe("SidebarFriendsList", () => {
  it("formats active game time without leading zeroes", () => {
    expect(
      formatGameDuration("2026-07-22T10:00:00.000Z", Date.parse("2026-07-22T10:04:00.000Z")),
    ).toBe("4m");
    expect(
      formatGameDuration("2026-07-22T10:00:00.000Z", Date.parse("2026-07-22T11:24:00.000Z")),
    ).toBe("1h 24m");
  });

  it("shows skeleton rows while friends are loading", () => {
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList friends={[]} isLoading canInvite={false} />
      </SidebarProvider>,
    );

    expect(view.container.querySelector('[aria-busy="true"]')).toBeTruthy();
    expect(view.container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(3);
    const skeletonClassName = view.container.querySelector('[data-slot="skeleton"]')?.className;
    expect(skeletonClassName).toContain("w-full");
    expect(skeletonClassName).toContain("group-data-[collapsible=icon]:size-8");
    expect(view.queryByText("No friends yet")).toBeNull();
  });

  it("keeps avatar menu triggers available when the sidebar collapses", () => {
    const view = render(
      <SidebarProvider defaultOpen={false}>
        <SidebarFriendsList
          friends={[{ id: "friend-1", name: "Devon", online: true }]}
          canInvite={false}
        />
      </SidebarProvider>,
    );

    const group = view.container.querySelector('[data-slot="sidebar-group"]');
    const trigger = view.getByRole("button", { name: "Open menu for Devon" });

    expect(group?.className).not.toContain("group-data-[collapsible=icon]:hidden");
    expect(trigger.className).toContain("group-data-[collapsible=icon]:p-1!");
    expect(trigger.querySelector('[data-slot="avatar"]')).toBeTruthy();
  });

  it("opens a friend menu with invite, profile, and unfriend actions", async () => {
    const onInvite = vi.fn(async () => undefined);
    const onUnfriend = vi.fn(async () => undefined);
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList
          friends={[
            { id: "friend-1", name: "Devon", online: true },
            { id: "friend-2", name: "Emery", online: false },
          ]}
          canInvite
          onInvite={onInvite}
          onUnfriend={onUnfriend}
        />
      </SidebarProvider>,
    );

    expect(view.getByText("Devon")).toBeTruthy();
    expect(view.getByText("Emery")).toBeTruthy();
    fireEvent.click(view.getByRole("button", { name: "Open menu for Devon" }));

    const inviteItem = await view.findByRole("menuitem", { name: "Invite to game" });
    expect(inviteItem.getAttribute("aria-disabled")).not.toBe("true");
    expect(view.getByRole("menuitem", { name: "View profile" })).toBeTruthy();
    fireEvent.click(inviteItem);

    await waitFor(() => expect(onInvite).toHaveBeenCalledWith("friend-1"));

    fireEvent.click(view.getByRole("button", { name: "Open menu for Devon" }));
    fireEvent.click(await view.findByRole("menuitem", { name: "Unfriend" }));
    await waitFor(() => expect(onUnfriend).toHaveBeenCalledWith("friend-1"));
  });

  it("disables game invites for offline friends", async () => {
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList
          friends={[{ id: "friend-1", name: "Emery", online: false }]}
          canInvite
          onInvite={vi.fn(async () => undefined)}
        />
      </SidebarProvider>,
    );

    fireEvent.click(view.getByRole("button", { name: "Open menu for Emery" }));

    expect(
      (await view.findByRole("menuitem", { name: "Invite to game" })).getAttribute("aria-disabled"),
    ).toBe("true");
  });

  it("shows active game duration and lets the viewer spectate", async () => {
    const onSpectate = vi.fn(async () => undefined);
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList
          friends={[
            {
              id: "friend-1",
              name: "Devon",
              online: true,
              activeGame: { startedAt: new Date(Date.now() - 84 * 60_000).toISOString() },
            },
          ]}
          canInvite={false}
          onSpectate={onSpectate}
        />
      </SidebarProvider>,
    );

    expect(view.container.querySelector('[aria-label="In game · 1h 24m"]')).toBeTruthy();
    const friendButton = view.getByRole("button", { name: "Open menu for Devon" });
    expect(friendButton.getAttribute("data-size")).toBe("default");
    fireEvent.click(friendButton);
    fireEvent.click(await view.findByRole("menuitem", { name: "Watch game" }));
    await waitFor(() => expect(onSpectate).toHaveBeenCalledWith("friend-1"));
  });
});

// @vitest-environment jsdom
import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vite-plus/test";
import { SidebarFriendsList } from "#/components/social/friends-list";
import { SidebarProvider } from "#/components/ui/sidebar";

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation(() => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })),
});

describe("SidebarFriendsList", () => {
  it("shows skeleton rows while friends are loading", () => {
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList friends={[]} isLoading canInvite={false} />
      </SidebarProvider>,
    );

    expect(view.container.querySelector('[aria-busy="true"]')).toBeTruthy();
    expect(view.container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(6);
    expect(view.queryByText("No friends yet")).toBeNull();
  });

  it("shows friends and invites an available online friend", async () => {
    const onInvite = vi.fn(async () => undefined);
    const view = render(
      <SidebarProvider>
        <SidebarFriendsList
          friends={[
            { id: "friend-1", name: "Devon", online: true },
            { id: "friend-2", name: "Emery", online: false },
          ]}
          canInvite
          onInvite={onInvite}
        />
      </SidebarProvider>,
    );

    expect(view.getByText("Devon")).toBeTruthy();
    expect(view.getByText("Emery")).toBeTruthy();
    fireEvent.click(view.getByRole("button", { name: "Invite to game" }));

    await waitFor(() => expect(onInvite).toHaveBeenCalledWith("friend-1"));
  });
});

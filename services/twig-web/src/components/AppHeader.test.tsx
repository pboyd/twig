import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppHeader } from "./AppHeader";

const mockNavigate = vi.fn();
vi.mock("react-router", async (importActual) => {
  const actual = await importActual<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const mockClear = vi.fn();
vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({ clear: mockClear })),
  };
});

function renderHeader(initialPath = "/tasks") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <AppHeader />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

/** Returns the open compact-menu dropdown element (call only after opening the menu). */
function getMenuDropdown() {
  return document.getElementById("header-menu")!;
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true }));
});

// --- US1: Primary nav ---

describe("AppHeader — US1: primary navigation", () => {
  it("renders Tasks link", () => {
    renderHeader();
    expect(screen.getByRole("link", { name: /tasks/i })).toBeTruthy();
  });

  it("renders Plan link", () => {
    renderHeader();
    expect(screen.getByRole("link", { name: /plan/i })).toBeTruthy();
  });

  it("renders Goals link", () => {
    renderHeader();
    expect(screen.getByRole("link", { name: /goals/i })).toBeTruthy();
  });

  it("Tasks link has active styling when on /tasks", () => {
    renderHeader("/tasks");
    const tasksLink = screen.getByRole("link", { name: /tasks/i });
    expect(tasksLink.className).toMatch(/bg-gray/);
  });

  it("Plan link has active styling when on /plan", () => {
    renderHeader("/plan");
    const planLink = screen.getByRole("link", { name: /plan/i });
    expect(planLink.className).toMatch(/bg-gray/);
  });

  it("logo links to /tasks with an accessible name", () => {
    renderHeader();
    const homeLink = screen.getByRole("link", { name: /twig|home/i });
    expect(homeLink.getAttribute("href")).toBe("/tasks");
  });
});

// --- US2: Compact menu ---

describe("AppHeader — US2: compact menu trigger", () => {
  it("renders a menu trigger button with an accessible name", () => {
    renderHeader();
    expect(screen.getByRole("button", { name: /menu/i })).toBeTruthy();
  });

  it("menu trigger has aria-controls and aria-expanded (disclosure pattern)", () => {
    renderHeader();
    const trigger = screen.getByRole("button", { name: /menu/i });
    expect(trigger.getAttribute("aria-controls")).toBe("header-menu");
    expect(trigger.hasAttribute("aria-expanded")).toBe(true);
    // disclosure pattern: no aria-haspopup="menu"
    expect(trigger.getAttribute("aria-haspopup")).toBeNull();
  });

  it("menu trigger starts with aria-expanded=false", () => {
    renderHeader();
    const trigger = screen.getByRole("button", { name: /menu/i });
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
  });

  it("opening the menu sets aria-expanded=true", () => {
    renderHeader();
    const trigger = screen.getByRole("button", { name: /menu/i });
    fireEvent.click(trigger);
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
  });

  it("opening the menu reveals Account link", () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    expect(within(getMenuDropdown()).getByRole("link", { name: /account/i })).toBeTruthy();
  });

  it("opening the menu reveals Download link", () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    expect(within(getMenuDropdown()).getByRole("link", { name: /download/i })).toBeTruthy();
  });

  it("opening the menu reveals Sign out item", () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    expect(within(getMenuDropdown()).getByRole("button", { name: /sign out/i })).toBeTruthy();
  });

  it("clicking Account closes the menu", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    fireEvent.click(within(getMenuDropdown()).getByRole("link", { name: /account/i }));
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /menu/i }).getAttribute("aria-expanded")
      ).toBe("false")
    );
  });

  it("clicking Download closes the menu", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    fireEvent.click(within(getMenuDropdown()).getByRole("link", { name: /download/i }));
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /menu/i }).getAttribute("aria-expanded")
      ).toBe("false")
    );
  });

  it("pressing Escape closes the menu", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /menu/i }).getAttribute("aria-expanded")
      ).toBe("false")
    );
  });

  it("pressing Escape returns focus to the menu trigger", async () => {
    renderHeader();
    const trigger = screen.getByRole("button", { name: /menu/i });
    fireEvent.click(trigger);
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  it("Sign out (in menu) posts to /auth/logout and navigates to /login", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    fireEvent.click(within(getMenuDropdown()).getByRole("button", { name: /sign out/i }));
    await waitFor(() => expect(mockClear).toHaveBeenCalled());
    expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    expect(vi.mocked(fetch)).toHaveBeenCalledWith("/auth/logout", {
      method: "POST",
      credentials: "include",
    });
  });

  it("Sign out (in menu) closes the menu", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    fireEvent.click(within(getMenuDropdown()).getByRole("button", { name: /sign out/i }));
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /menu/i }).getAttribute("aria-expanded")
      ).toBe("false")
    );
  });
});

// --- US2: active state in menu ---

describe("AppHeader — US2: active state in menu", () => {
  it("Account link has active styling when on /account and menu is open", () => {
    renderHeader("/account");
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    const accountItem = within(getMenuDropdown()).getByRole("link", { name: /account/i });
    expect(accountItem.className).toMatch(/bg-gray/);
  });

  it("Download link has active styling when on /download and menu is open", () => {
    renderHeader("/download");
    fireEvent.click(screen.getByRole("button", { name: /menu/i }));
    const downloadItem = within(getMenuDropdown()).getByRole("link", { name: /download/i });
    expect(downloadItem.className).toMatch(/bg-gray/);
  });
});

// --- US3: Wide layout structure (C5/C6) ---

describe("AppHeader — US3: wide layout structure", () => {
  it("Download link is present in the header for wide layout", () => {
    renderHeader();
    const links = screen.getAllByRole("link", { name: /download/i });
    expect(links.length).toBeGreaterThan(0);
    // wide-layout link has hidden sm:block gating
    expect(links.some((l) => l.className.includes("hidden") && l.className.includes("sm:block"))).toBe(true);
  });

  it("Account link is present in the header for wide layout", () => {
    renderHeader();
    const links = screen.getAllByRole("link", { name: /account/i });
    expect(links.length).toBeGreaterThan(0);
    expect(links.some((l) => l.className.includes("hidden") && l.className.includes("sm:block"))).toBe(true);
  });

  it("Sign out button is present in the header for wide layout (C5)", () => {
    renderHeader();
    // Before menu is open, the only Sign out in the DOM is the desktop button
    const desktopSignOut = screen.getByRole("button", { name: /sign out/i });
    // Must be gated hidden (small) / visible (sm+)
    expect(desktopSignOut.className).toContain("hidden");
    expect(desktopSignOut.className).toContain("sm:inline-flex");
  });

  it("Sign out (desktop) posts to /auth/logout and navigates to /login (C13)", async () => {
    renderHeader();
    fireEvent.click(screen.getByRole("button", { name: /sign out/i }));
    await waitFor(() => expect(mockClear).toHaveBeenCalled());
    expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    expect(vi.mocked(fetch)).toHaveBeenCalledWith("/auth/logout", {
      method: "POST",
      credentials: "include",
    });
  });

  it("compact menu trigger is gated to small viewports only (C6)", () => {
    renderHeader();
    const trigger = screen.getByRole("button", { name: /menu/i });
    // The trigger's immediate parent wrapper div in AppHeader has sm:hidden
    const wrapper = trigger.closest('[class*="sm:hidden"]');
    expect(wrapper).toBeTruthy();
  });
});

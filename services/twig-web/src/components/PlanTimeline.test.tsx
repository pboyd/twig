import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { PlanTimeline } from "./PlanTimeline";
import type { ResolvedEntry } from "../lib/planView";

function makeEntry(overrides: Partial<ResolvedEntry> = {}): ResolvedEntry {
  return {
    id: 1,
    displayName: "Test entry",
    kind: "task",
    taskId: 1n,
    completed: false,
    timed: true,
    startMinute: 540,
    endMinute: 570,
    durationMinute: 30,
    ...overrides,
  };
}

function renderTimeline(
  entries: ResolvedEntry[],
  overrides: Partial<Parameters<typeof PlanTimeline>[0]> = {}
) {
  return render(
    <MemoryRouter>
      <PlanTimeline
        entries={entries}
        day="2026-07-02"
        onToggleComplete={vi.fn()}
        onRemove={vi.fn()}
        pendingIds={new Set()}
        entryErrors={new Map()}
        {...overrides}
      />
    </MemoryRouter>
  );
}

describe("PlanTimeline", () => {
  it("renders hour labels in the gutter", () => {
    const entries = [makeEntry({ startMinute: 540, endMinute: 570 })];
    renderTimeline(entries, { day: "2026-07-02" });
    expect(screen.getByText("8:00 am")).toBeTruthy();
    expect(screen.getByText("9:00 am")).toBeTruthy();
  });

  it("renders entry block with name", () => {
    const entries = [makeEntry({ displayName: "Standup", startMinute: 540, endMinute: 570 })];
    renderTimeline(entries);
    expect(screen.getByText("Standup")).toBeTruthy();
  });

  it("renders entry block spanning proportional rows (60 min = 4 slots)", () => {
    const entries = [makeEntry({ startMinute: 540, endMinute: 600 })];
    const { container } = renderTimeline(entries);
    const entryBlock = container.querySelector('[class*="col-start-2"][class*="flex"]');
    expect(entryBlock).toBeTruthy();
    const style = entryBlock?.getAttribute("style") || "";
    const spanMatch = style.match(/span\s+(\d+)/);
    expect(spanMatch).toBeTruthy();
    expect(Number(spanMatch![1])).toBeGreaterThanOrEqual(4);
  });

  it("renders task entry as link", () => {
    const entries = [makeEntry({ kind: "task", taskId: 42n, displayName: "My task" })];
    renderTimeline(entries);
    const link = screen.getByRole("link", { name: /my task/i });
    expect(link.getAttribute("href")).toBe("/tasks/42");
  });

  it("renders event as italic text (not a link)", () => {
    const entries = [makeEntry({ kind: "event", taskId: undefined, displayName: "Lunch" })];
    renderTimeline(entries);
    expect(screen.queryByRole("link", { name: /lunch/i })).toBeNull();
    expect(screen.getByText("Lunch")).toBeTruthy();
  });

  it("applies strikethrough for completed entries", () => {
    const entries = [makeEntry({ completed: true, displayName: "Done task" })];
    renderTimeline(entries);
    const link = screen.getByRole("link", { name: /done task/i });
    expect(link.className).toContain("line-through");
  });

  it("shows remove button and fires onRemove", () => {
    const onRemove = vi.fn();
    const entries = [makeEntry({ id: 7 })];
    renderTimeline(entries, { onRemove });
    fireEvent.click(screen.getByTestId("remove-btn"));
    expect(onRemove).toHaveBeenCalledWith(expect.objectContaining({ id: 7 }));
  });

  it("shows per-entry error text", () => {
    const entries = [makeEntry({ id: 1 })];
    renderTimeline(entries, { entryErrors: new Map([[1, "Something went wrong"]]) });
    expect(screen.getByText("Something went wrong")).toBeTruthy();
  });

  it("gap rows (no entry) render empty", () => {
    const entries = [
      makeEntry({ id: 1, startMinute: 540, endMinute: 570 }),
      makeEntry({ id: 2, startMinute: 600, endMinute: 660 }),
    ];
    const { container } = renderTimeline(entries);
    const grid = container.querySelector(".grid");
    expect(grid).toBeTruthy();
  });
});

describe("PlanTimeline — now indicator (US3)", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows now-indicator when day is today and now is within the window", () => {
    const today = new Date();
    const y = today.getFullYear();
    const m = String(today.getMonth() + 1).padStart(2, "0");
    const d = String(today.getDate()).padStart(2, "0");
    const dayStr = `${y}-${m}-${d}`;
    // Set time to 10:00 = 600 minutes
    today.setHours(10, 0, 0, 0);
    vi.setSystemTime(today);

    const entries = [makeEntry({ startMinute: 540, endMinute: 570 })];
    renderTimeline(entries, { day: dayStr });
    expect(screen.getByTestId("now-indicator")).toBeTruthy();
  });

  it("does not show now-indicator on a non-today day", () => {
    const today = new Date();
    today.setHours(10, 0, 0, 0);
    vi.setSystemTime(today);

    const entries = [makeEntry({ startMinute: 540, endMinute: 570 })];
    renderTimeline(entries, { day: "2026-01-01" });
    expect(screen.queryByTestId("now-indicator")).toBeNull();
  });

  it("does not show now-indicator when now is outside window", () => {
    const today = new Date();
    const y = today.getFullYear();
    const m = String(today.getMonth() + 1).padStart(2, "0");
    const d = String(today.getDate()).padStart(2, "0");
    const dayStr = `${y}-${m}-${d}`;
    // Set time to 22:00 = 1320 minutes (outside default 480-1020)
    today.setHours(22, 0, 0, 0);
    vi.setSystemTime(today);

    const entries = [makeEntry({ startMinute: 540, endMinute: 570 })];
    renderTimeline(entries, { day: dayStr });
    expect(screen.queryByTestId("now-indicator")).toBeNull();
  });
});

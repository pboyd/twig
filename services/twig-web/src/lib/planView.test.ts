import { describe, it, expect } from "vitest";
import {
  todayString,
  addDays,
  isToday,
  formatDayLabel,
  formatMinute,
  formatTimeRange,
  resolveEntries,
  groupPlan,
} from "./planView";
import type { PlanEntry } from "../gen/plan/v1/plan_pb";

function makeEntry(overrides: Partial<PlanEntry> = {}): PlanEntry {
  return {
    day: "2026-06-06",
    id: 1,
    taskId: 0n,
    name: "",
    startMinute: undefined,
    durationMinute: 60,
    completed: false,
    ...overrides,
  } as unknown as PlanEntry;
}

describe("todayString", () => {
  it("returns today as YYYY-MM-DD", () => {
    const result = todayString();
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("matches new Date() local date", () => {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    expect(todayString()).toBe(`${y}-${m}-${day}`);
  });
});

describe("addDays", () => {
  it("adds positive delta", () => {
    expect(addDays("2026-06-06", 1)).toBe("2026-06-07");
  });

  it("subtracts with negative delta", () => {
    expect(addDays("2026-06-06", -1)).toBe("2026-06-05");
  });

  it("handles month rollover forward", () => {
    expect(addDays("2026-06-30", 1)).toBe("2026-07-01");
  });

  it("handles month rollover backward", () => {
    expect(addDays("2026-07-01", -1)).toBe("2026-06-30");
  });

  it("handles year rollover forward", () => {
    expect(addDays("2026-12-31", 1)).toBe("2027-01-01");
  });

  it("handles year rollover backward", () => {
    expect(addDays("2027-01-01", -1)).toBe("2026-12-31");
  });

  it("zero delta returns same day", () => {
    expect(addDays("2026-06-06", 0)).toBe("2026-06-06");
  });
});

describe("isToday", () => {
  it("returns true for today", () => {
    expect(isToday(todayString())).toBe(true);
  });

  it("returns false for yesterday", () => {
    expect(isToday(addDays(todayString(), -1))).toBe(false);
  });

  it("returns false for tomorrow", () => {
    expect(isToday(addDays(todayString(), 1))).toBe(false);
  });
});

describe("formatDayLabel", () => {
  it("includes 'Today' prefix for today", () => {
    const label = formatDayLabel(todayString());
    expect(label).toMatch(/^Today/);
  });

  it("does not include Today prefix for other days", () => {
    const label = formatDayLabel("2026-01-01");
    expect(label).not.toMatch(/^Today/);
  });

  it("includes day of week abbreviation", () => {
    // 2026-06-06 is a Saturday
    const label = formatDayLabel("2026-06-06");
    expect(label).toMatch(/Sat/);
  });
});

describe("formatMinute", () => {
  it("maps 0 to 12:00 am", () => {
    expect(formatMinute(0)).toBe("12:00 am");
  });

  it("maps 720 to 12:00 pm", () => {
    expect(formatMinute(720)).toBe("12:00 pm");
  });

  it("maps 1439 to 11:59 pm", () => {
    expect(formatMinute(1439)).toBe("11:59 pm");
  });

  it("maps 60 to 1:00 am", () => {
    expect(formatMinute(60)).toBe("1:00 am");
  });

  it("maps 540 to 9:00 am", () => {
    expect(formatMinute(540)).toBe("9:00 am");
  });

  it("maps 780 to 1:00 pm", () => {
    expect(formatMinute(780)).toBe("1:00 pm");
  });

  it("maps 1380 to 11:00 pm", () => {
    expect(formatMinute(1380)).toBe("11:00 pm");
  });
});

describe("formatTimeRange", () => {
  it("formats a time range", () => {
    expect(formatTimeRange(540, 60)).toBe("9:00 am – 10:00 am");
  });

  it("crosses noon boundary", () => {
    expect(formatTimeRange(660, 120)).toBe("11:00 am – 1:00 pm");
  });
});

describe("resolveEntries", () => {
  it("uses entry name override when provided", () => {
    const entry = makeEntry({ name: "My Override", taskId: 1n });
    const result = resolveEntries([entry], new Map([[1n, "Task Name"]]));
    expect(result[0].displayName).toBe("My Override");
  });

  it("falls back to task name when entry name is empty", () => {
    const entry = makeEntry({ name: "", taskId: 1n });
    const result = resolveEntries([entry], new Map([[1n, "Task Name"]]));
    expect(result[0].displayName).toBe("Task Name");
  });

  it("falls back to Untitled entry when name and task name are absent", () => {
    const entry = makeEntry({ name: "", taskId: 1n });
    const result = resolveEntries([entry], new Map());
    expect(result[0].displayName).toBe("Untitled entry");
  });

  it("uses Untitled entry for events with no name", () => {
    const entry = makeEntry({ name: "", taskId: 0n });
    const result = resolveEntries([entry], new Map());
    expect(result[0].displayName).toBe("Untitled entry");
  });

  it("sets kind=task for non-zero taskId", () => {
    const entry = makeEntry({ taskId: 42n });
    const result = resolveEntries([entry], new Map());
    expect(result[0].kind).toBe("task");
    expect(result[0].taskId).toBe(42n);
  });

  it("sets kind=event for zero taskId", () => {
    const entry = makeEntry({ taskId: 0n });
    const result = resolveEntries([entry], new Map());
    expect(result[0].kind).toBe("event");
    expect(result[0].taskId).toBeUndefined();
  });

  it("sets completed from entry", () => {
    const entry = makeEntry({ completed: true, taskId: 1n });
    const result = resolveEntries([entry], new Map());
    expect(result[0].completed).toBe(true);
  });

  it("sets timed=true when startMinute is present", () => {
    const entry = makeEntry({ startMinute: 540, durationMinute: 60 });
    const result = resolveEntries([entry], new Map());
    expect(result[0].timed).toBe(true);
    expect(result[0].startMinute).toBe(540);
    expect(result[0].endMinute).toBe(600);
    expect(result[0].timeLabel).toBe("9:00 am – 10:00 am");
  });

  it("sets timed=false when startMinute is absent", () => {
    const entry = makeEntry({ startMinute: undefined });
    const result = resolveEntries([entry], new Map());
    expect(result[0].timed).toBe(false);
    expect(result[0].startMinute).toBeUndefined();
    expect(result[0].endMinute).toBeUndefined();
    expect(result[0].timeLabel).toBeUndefined();
  });
});

describe("groupPlan", () => {
  it("splits timed and untimed entries", () => {
    const timed = makeEntry({ startMinute: 540, durationMinute: 60 });
    const untimed = makeEntry({ startMinute: undefined, id: 2 });
    const entries = resolveEntries([timed, untimed], new Map());
    const grouped = groupPlan(entries);
    expect(grouped.timed).toHaveLength(1);
    expect(grouped.untimed).toHaveLength(1);
    expect(grouped.timed[0].startMinute).toBe(540);
  });

  it("isEmpty is true when no entries", () => {
    const grouped = groupPlan([]);
    expect(grouped.isEmpty).toBe(true);
  });

  it("isEmpty is false when there are timed entries", () => {
    const timed = makeEntry({ startMinute: 540, durationMinute: 60 });
    const grouped = groupPlan(resolveEntries([timed], new Map()));
    expect(grouped.isEmpty).toBe(false);
  });

  it("isEmpty is false when there are only untimed entries", () => {
    const untimed = makeEntry({ startMinute: undefined });
    const grouped = groupPlan(resolveEntries([untimed], new Map()));
    expect(grouped.isEmpty).toBe(false);
  });

  // T005: completed untimed entries should be excluded from the untimed group
  it("excludes completed untimed entries from untimed group", () => {
    const completedUntimed = makeEntry({ id: 1, taskId: 1n, completed: true, startMinute: undefined });
    const incompleteUntimed = makeEntry({ id: 2, taskId: 2n, completed: false, startMinute: undefined });
    const entries = resolveEntries([completedUntimed, incompleteUntimed], new Map());
    const grouped = groupPlan(entries);
    expect(grouped.untimed).toHaveLength(1);
    expect(grouped.untimed[0].id).toBe(2);
  });

  it("keeps completed timed entries in timed group", () => {
    const completedTimed = makeEntry({ id: 1, taskId: 1n, completed: true, startMinute: 540, durationMinute: 60 });
    const entries = resolveEntries([completedTimed], new Map());
    const grouped = groupPlan(entries);
    expect(grouped.timed).toHaveLength(1);
    expect(grouped.timed[0].id).toBe(1);
  });

  it("isEmpty is true when the only untimed entries are completed and no timed entries", () => {
    const completedUntimed = makeEntry({ id: 1, taskId: 1n, completed: true, startMinute: undefined });
    const entries = resolveEntries([completedUntimed], new Map());
    const grouped = groupPlan(entries);
    expect(grouped.untimed).toHaveLength(0);
    expect(grouped.isEmpty).toBe(true);
  });

  // T013: reopened (incomplete) untimed entry should be in untimed group
  it("includes incomplete (reopened) untimed entries in untimed group", () => {
    const incompleteUntimed = makeEntry({ id: 1, taskId: 1n, completed: false, startMinute: undefined });
    const entries = resolveEntries([incompleteUntimed], new Map());
    const grouped = groupPlan(entries);
    expect(grouped.untimed).toHaveLength(1);
    expect(grouped.untimed[0].id).toBe(1);
  });
});

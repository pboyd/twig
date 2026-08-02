import { describe, it, expect } from "vitest";
import { isSnoozed } from "./snooze";
import type { Task } from "../gen/task/v1/task_pb";

function makeTask(snoozeSeconds?: bigint): Task {
  return {
    id: 1n,
    name: "Task",
    description: "",
    parentId: undefined,
    due: undefined,
    completedAt: undefined,
    estimate: 0,
    snoozeUntil: snoozeSeconds !== undefined ? { seconds: snoozeSeconds, nanos: 0 } : undefined,
  } as unknown as Task;
}

// Reference date: local noon on 2026-06-06 (getDate()=6 in any timezone)
const TODAY = new Date(2026, 5, 6, 12, 0, 0);
const TOMORROW_SECONDS = 1780790400n; // 2026-06-07T00:00:00Z
const TODAY_SECONDS = 1780704000n; // 2026-06-06T00:00:00Z
const YESTERDAY_SECONDS = 1780617600n; // 2026-06-05T00:00:00Z
const NEXT_MONTH_SECONDS = 1783382400n; // 2026-07-05T00:00:00Z
const NEXT_YEAR_SECONDS = 1812153600n; // 2027-06-01T00:00:00Z

describe("isSnoozed", () => {
  it("returns false when there is no task", () => {
    expect(isSnoozed(undefined, TODAY)).toBe(false);
  });

  it("returns false when snoozeUntil is unset", () => {
    expect(isSnoozed(makeTask(), TODAY)).toBe(false);
  });

  it("returns true for a day strictly after today", () => {
    expect(isSnoozed(makeTask(TOMORROW_SECONDS), TODAY)).toBe(true);
  });

  it("returns false for today (not strictly after)", () => {
    expect(isSnoozed(makeTask(TODAY_SECONDS), TODAY)).toBe(false);
  });

  it("returns false for a past day", () => {
    expect(isSnoozed(makeTask(YESTERDAY_SECONDS), TODAY)).toBe(false);
  });

  it("returns true for a future month", () => {
    expect(isSnoozed(makeTask(NEXT_MONTH_SECONDS), TODAY)).toBe(true);
  });

  it("returns true for a future year", () => {
    expect(isSnoozed(makeTask(NEXT_YEAR_SECONDS), TODAY)).toBe(true);
  });

  it("defaults refDate to now", () => {
    const future = new Date();
    future.setUTCFullYear(future.getUTCFullYear() + 5);
    expect(isSnoozed(makeTask(BigInt(Math.floor(future.getTime() / 1000))))).toBe(true);
  });
});

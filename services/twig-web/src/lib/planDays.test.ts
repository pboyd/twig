import { describe, it, expect, vi, afterEach } from "vitest";
import { todayIso, tomorrowIso, dayPickerLabel } from "./planDays";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("todayIso", () => {
  it("returns a YYYY-MM-DD string", () => {
    const result = todayIso();
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("matches local date", () => {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    expect(todayIso()).toBe(`${y}-${m}-${day}`);
  });
});

describe("tomorrowIso", () => {
  it("returns a date one day after todayIso", () => {
    const today = todayIso();
    const tomorrow = tomorrowIso();
    const [y, m, d] = today.split("-").map(Number);
    const expected = new Date(y, m - 1, d + 1);
    const ey = expected.getFullYear();
    const em = String(expected.getMonth() + 1).padStart(2, "0");
    const ed = String(expected.getDate()).padStart(2, "0");
    expect(tomorrow).toBe(`${ey}-${em}-${ed}`);
  });
});

describe("dayPickerLabel", () => {
  it("returns 'Today' for today's ISO string", () => {
    expect(dayPickerLabel(todayIso())).toBe("Today");
  });

  it("returns 'Tomorrow' for tomorrow's ISO string", () => {
    expect(dayPickerLabel(tomorrowIso())).toBe("Tomorrow");
  });

  it("returns a formatted label for other dates", () => {
    const label = dayPickerLabel("2025-01-15");
    expect(label).not.toBe("Today");
    expect(label).not.toBe("Tomorrow");
    expect(typeof label).toBe("string");
    expect(label.length).toBeGreaterThan(0);
  });
});

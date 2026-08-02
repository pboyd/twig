import { describe, it, expect } from "vitest";
import { formatTimestamp, formatDueDate } from "./formatTimestamp";

function ts(seconds: bigint, nanos: number = 0) {
  return { seconds, nanos, $typeName: "google.protobuf.Timestamp" as const };
}

describe("formatTimestamp", () => {
  it("formats a known timestamp", () => {
    const t = ts(1719000000n);
    const result = formatTimestamp(t);
    expect(result).toMatch(/2024/);
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });

  it("returns empty string for undefined", () => {
    expect(formatTimestamp(undefined)).toBe("");
  });

  it("includes time in the output", () => {
    const t = ts(1719000000n);
    const result = formatTimestamp(t);
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });
});

describe("formatDueDate", () => {
  it("formats a date-only timestamp", () => {
    const t = ts(1719000000n);
    const result = formatDueDate(t);
    expect(result).toMatch(/2024/);
  });

  it("returns empty string for undefined", () => {
    expect(formatDueDate(undefined)).toBe("");
  });

  it("does not include time", () => {
    const t = ts(1719000000n);
    const result = formatDueDate(t);
    expect(result).not.toMatch(/\d{1,2}:\d{2}/);
  });

  it("renders the UTC calendar day regardless of local timezone", () => {
    // 2026-08-03T00:00:00Z — in America/New_York (UTC-4/-5) this instant is
    // still Aug 2 locally, but the day is pinned to UTC and must render as 3.
    const t = ts(BigInt(Date.UTC(2026, 7, 3) / 1000));
    const result = formatDueDate(t);
    expect(result).toContain("3");
    expect(result).not.toContain("2,");
  });
});

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
});

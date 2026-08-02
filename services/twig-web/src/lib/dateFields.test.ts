import { describe, it, expect } from "vitest";
import { isoDayToTimestamp, timestampToIsoDay, resolveDayEdit } from "./dateFields";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

describe("isoDayToTimestamp", () => {
  it("converts an ISO day to midnight UTC seconds with zero nanos", () => {
    const ts = isoDayToTimestamp("2026-08-02");
    expect(ts).toBeDefined();
    expect(ts!.seconds).toBe(BigInt(Date.UTC(2026, 7, 2) / 1000));
    expect(ts!.nanos).toBe(0);
  });

  it("converts past dates without complaint", () => {
    const ts = isoDayToTimestamp("2000-01-01");
    expect(ts).toBeDefined();
    expect(ts!.seconds).toBe(BigInt(Date.UTC(2000, 0, 1) / 1000));
  });

  it("returns undefined for an empty string", () => {
    expect(isoDayToTimestamp("")).toBeUndefined();
  });

  it("returns undefined for undefined input", () => {
    expect(isoDayToTimestamp(undefined)).toBeUndefined();
  });
});

describe("timestampToIsoDay", () => {
  it("converts a midnight-UTC timestamp to an ISO day string", () => {
    const ts: Timestamp = {
      seconds: BigInt(Date.UTC(2026, 7, 2) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp",
    };
    expect(timestampToIsoDay(ts)).toBe("2026-08-02");
  });

  it("returns an empty string for undefined", () => {
    expect(timestampToIsoDay(undefined)).toBe("");
  });
});

describe("dateFields round-trip", () => {
  it("is lossless day -> timestamp -> day", () => {
    const day = "2026-08-02";
    expect(timestampToIsoDay(isoDayToTimestamp(day))).toBe(day);
  });

  it("is lossless empty -> undefined -> empty", () => {
    expect(timestampToIsoDay(isoDayToTimestamp(""))).toBe("");
  });
});

describe("resolveDayEdit", () => {
  it("keeps the original timestamp (including time-of-day) when the day is unchanged", () => {
    const original: Timestamp = {
      seconds: BigInt(Date.UTC(2026, 7, 3, 17, 0, 0) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp",
    };
    expect(resolveDayEdit(original, "2026-08-03")).toBe(original);
  });

  it("converts to midnight UTC when the day changed", () => {
    const original: Timestamp = {
      seconds: BigInt(Date.UTC(2026, 7, 3, 17, 0, 0) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp",
    };
    const result = resolveDayEdit(original, "2026-09-01");
    expect(result).not.toBe(original);
    expect(result!.seconds).toBe(BigInt(Date.UTC(2026, 8, 1) / 1000));
  });

  it("returns undefined when the field is cleared", () => {
    const original: Timestamp = {
      seconds: BigInt(Date.UTC(2026, 7, 3) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp",
    };
    expect(resolveDayEdit(original, undefined)).toBeUndefined();
    expect(resolveDayEdit(original, "")).toBeUndefined();
  });

  it("keeps undefined as undefined when nothing was set and nothing was entered", () => {
    expect(resolveDayEdit(undefined, undefined)).toBeUndefined();
    expect(resolveDayEdit(undefined, "")).toBeUndefined();
  });

  it("sets a new timestamp when a day is entered where none existed before", () => {
    const result = resolveDayEdit(undefined, "2026-08-03");
    expect(result!.seconds).toBe(BigInt(Date.UTC(2026, 7, 3) / 1000));
  });
});

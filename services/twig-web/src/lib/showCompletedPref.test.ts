import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { readShowCompleted, writeShowCompleted } from "./showCompletedPref";

const STORAGE_KEY = "twig-show-completed";

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("readShowCompleted", () => {
  it("returns false when no key is stored", () => {
    expect(readShowCompleted()).toBe(false);
  });

  it("returns true after writing true", () => {
    localStorage.setItem(STORAGE_KEY, "true");
    expect(readShowCompleted()).toBe(true);
  });

  it("returns false for any non-'true' stored value", () => {
    for (const val of ["false", "1", "yes", "TRUE", ""]) {
      localStorage.setItem(STORAGE_KEY, val);
      expect(readShowCompleted()).toBe(false);
    }
  });

  it("returns false when localStorage throws", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("storage error");
    });
    expect(readShowCompleted()).toBe(false);
  });
});

describe("writeShowCompleted", () => {
  it("round-trips: write true then read returns true", () => {
    writeShowCompleted(true);
    expect(readShowCompleted()).toBe(true);
  });

  it("round-trips: write false then read returns false", () => {
    writeShowCompleted(true);
    writeShowCompleted(false);
    expect(readShowCompleted()).toBe(false);
  });

  it("swallows storage exceptions", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("storage full");
    });
    expect(() => writeShowCompleted(true)).not.toThrow();
  });
});

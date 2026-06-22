import { describe, it, expect, vi, beforeEach } from "vitest";
import { readShowHiddenGoals, writeShowHiddenGoals } from "./showHiddenGoalsPref";

const KEY = "twig-show-hidden-goals";

beforeEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
});

describe("readShowHiddenGoals", () => {
  it("returns false by default", () => {
    expect(readShowHiddenGoals()).toBe(false);
  });

  it("returns true when localStorage is 'true'", () => {
    localStorage.setItem(KEY, "true");
    expect(readShowHiddenGoals()).toBe(true);
  });

  it("returns false when localStorage throws", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("denied");
    });
    expect(readShowHiddenGoals()).toBe(false);
  });
});

describe("writeShowHiddenGoals", () => {
  it("persists the value", () => {
    writeShowHiddenGoals(true);
    expect(localStorage.getItem(KEY)).toBe("true");
  });

  it("survives localStorage exceptions", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("full");
    });
    expect(() => writeShowHiddenGoals(true)).not.toThrow();
  });
});

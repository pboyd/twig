import { describe, it, expect } from "vitest";
import { reorderAnchor } from "./reorderAnchor";

const siblings = [1n, 2n, 3n, 4n, 5n];

describe("reorderAnchor", () => {
  it("returns null for drop-on-self", () => {
    expect(reorderAnchor(2n, 2n, siblings)).toBeNull();
  });

  it("returns beforeTaskId when moving up (activeIndex > overIndex)", () => {
    const result = reorderAnchor(4n, 2n, siblings);
    expect(result).toEqual({ case: "beforeTaskId", value: 2n });
  });

  it("returns afterTaskId when moving down (activeIndex < overIndex)", () => {
    const result = reorderAnchor(2n, 4n, siblings);
    expect(result).toEqual({ case: "afterTaskId", value: 4n });
  });

  it("returns beforeTaskId when moving to first position", () => {
    const result = reorderAnchor(5n, 1n, siblings);
    expect(result).toEqual({ case: "beforeTaskId", value: 1n });
  });

  it("returns afterTaskId when moving to last position", () => {
    const result = reorderAnchor(1n, 5n, siblings);
    expect(result).toEqual({ case: "afterTaskId", value: 5n });
  });

  it("returns null when activeId is not in siblings", () => {
    expect(reorderAnchor(99n, 2n, siblings)).toBeNull();
  });

  it("returns null when overId is not in siblings", () => {
    expect(reorderAnchor(2n, 99n, siblings)).toBeNull();
  });

  it("returns beforeTaskId for adjacent upward move", () => {
    const result = reorderAnchor(3n, 2n, siblings);
    expect(result).toEqual({ case: "beforeTaskId", value: 2n });
  });

  it("returns afterTaskId for adjacent downward move", () => {
    const result = reorderAnchor(3n, 4n, siblings);
    expect(result).toEqual({ case: "afterTaskId", value: 4n });
  });
});

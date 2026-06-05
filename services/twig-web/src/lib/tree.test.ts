import { describe, it, expect } from "vitest";
import { buildTree, findSiblingIds } from "./tree";
import type { Task } from "../gen/task/v1/task_pb";

function makeTask(id: bigint, parentId?: bigint, position?: bigint): Task {
  return {
    id,
    name: `Task ${id}`,
    description: "",
    parentId,
    due: undefined,
    completedAt: undefined,
    estimate: 0,
    position: position ?? 0n,
  } as unknown as Task;
}

describe("buildTree", () => {
  it("returns empty array for empty input", () => {
    expect(buildTree([])).toEqual([]);
  });

  it("returns top-level tasks as roots", () => {
    const tasks = [makeTask(1n), makeTask(2n)];
    const tree = buildTree(tasks);
    expect(tree).toHaveLength(2);
    expect(tree[0].task.id).toBe(1n);
    expect(tree[1].task.id).toBe(2n);
    expect(tree[0].depth).toBe(0);
    expect(tree[1].depth).toBe(0);
  });

  it("nests tasks by parent_id", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n)];
    const tree = buildTree(tasks);
    expect(tree).toHaveLength(1);
    expect(tree[0].task.id).toBe(1n);
    expect(tree[0].children).toHaveLength(1);
    expect(tree[0].children[0].task.id).toBe(2n);
    expect(tree[0].children[0].depth).toBe(1);
  });

  it("preserves id-ascending sibling order when positions are equal", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n), makeTask(3n, 1n), makeTask(4n, 1n)];
    const tree = buildTree(tasks);
    const children = tree[0].children;
    expect(children.map((n) => n.task.id)).toEqual([2n, 3n, 4n]);
  });

  it("sorts siblings by position ascending", () => {
    const tasks = [
      makeTask(1n, undefined, 0n),
      makeTask(2n, 1n, 2n),
      makeTask(3n, 1n, 0n),
      makeTask(4n, 1n, 1n),
    ];
    const tree = buildTree(tasks);
    expect(tree[0].children.map((n) => n.task.id)).toEqual([3n, 4n, 2n]);
  });

  it("uses id as tiebreaker when positions are equal", () => {
    const tasks = [
      makeTask(1n, undefined, 0n),
      makeTask(3n, 1n, 0n),
      makeTask(2n, 1n, 0n),
    ];
    const tree = buildTree(tasks);
    expect(tree[0].children.map((n) => n.task.id)).toEqual([2n, 3n]);
  });

  it("sorts root siblings by position", () => {
    const tasks = [makeTask(3n, undefined, 0n), makeTask(1n, undefined, 2n), makeTask(2n, undefined, 1n)];
    const tree = buildTree(tasks);
    expect(tree.map((n) => n.task.id)).toEqual([3n, 2n, 1n]);
  });

  it("handles multi-level nesting with correct depth", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n), makeTask(3n, 2n)];
    const tree = buildTree(tasks);
    expect(tree[0].depth).toBe(0);
    expect(tree[0].children[0].depth).toBe(1);
    expect(tree[0].children[0].children[0].depth).toBe(2);
  });

  it("treats orphans (missing parent) as roots", () => {
    const tasks = [makeTask(2n, 99n)];
    const tree = buildTree(tasks);
    expect(tree).toHaveLength(1);
    expect(tree[0].task.id).toBe(2n);
    expect(tree[0].depth).toBe(0);
  });
});

describe("findSiblingIds", () => {
  it("returns root ids when task is at root level", () => {
    const tasks = [makeTask(1n, undefined, 0n), makeTask(2n, undefined, 1n)];
    const tree = buildTree(tasks);
    expect(findSiblingIds(tree, 1n)).toEqual([1n, 2n]);
    expect(findSiblingIds(tree, 2n)).toEqual([1n, 2n]);
  });

  it("returns children ids when task is a child", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n, 0n), makeTask(3n, 1n, 1n)];
    const tree = buildTree(tasks);
    expect(findSiblingIds(tree, 2n)).toEqual([2n, 3n]);
    expect(findSiblingIds(tree, 3n)).toEqual([2n, 3n]);
  });

  it("returns null for unknown task id", () => {
    const tasks = [makeTask(1n)];
    const tree = buildTree(tasks);
    expect(findSiblingIds(tree, 99n)).toBeNull();
  });

  it("finds siblings at nested depth", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n), makeTask(3n, 2n, 0n), makeTask(4n, 2n, 1n)];
    const tree = buildTree(tasks);
    expect(findSiblingIds(tree, 3n)).toEqual([3n, 4n]);
  });
});

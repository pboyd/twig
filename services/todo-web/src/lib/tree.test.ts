import { describe, it, expect } from "vitest";
import { buildTree } from "./tree";
import type { Task } from "../gen/task/v1/task_pb";

function makeTask(id: bigint, parentId?: bigint): Task {
  return {
    id,
    name: `Task ${id}`,
    description: "",
    parentId,
    due: undefined,
    completedAt: undefined,
    estimate: 0,
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

  it("preserves id-ascending sibling order", () => {
    const tasks = [makeTask(1n), makeTask(2n, 1n), makeTask(3n, 1n), makeTask(4n, 1n)];
    const tree = buildTree(tasks);
    const children = tree[0].children;
    expect(children.map((n) => n.task.id)).toEqual([2n, 3n, 4n]);
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

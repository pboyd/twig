import { describe, it, expect } from "vitest";
import { filterTree } from "./filterTree";
import type { TaskNode } from "./tree";
import type { Task } from "../gen/task/v1/task_pb";

function makeTask(id: bigint, completed = false, parentId?: bigint): Task {
  return {
    id,
    name: `Task ${id}`,
    description: "",
    parentId,
    due: undefined,
    completedAt: completed ? { seconds: 1000n, nanos: 0 } : undefined,
    estimate: 0,
  } as unknown as Task;
}

function makeNode(task: Task, children: TaskNode[] = [], depth = 0): TaskNode {
  return { task, children, depth };
}

describe("filterTree", () => {
  it("identity passthrough when showCompleted is true", () => {
    const nodes = [
      makeNode(makeTask(1n, true)),
      makeNode(makeTask(2n, false)),
    ];
    const result = filterTree(nodes, true);
    expect(result).toEqual(nodes);
  });

  it("drops a completed leaf", () => {
    const nodes = [makeNode(makeTask(1n, true))];
    expect(filterTree(nodes, false)).toEqual([]);
  });

  it("keeps an incomplete leaf", () => {
    const node = makeNode(makeTask(1n, false));
    expect(filterTree([node], false)).toEqual([node]);
  });

  it("keeps incomplete parent, removes its completed children, preserves order", () => {
    const child1 = makeNode(makeTask(2n, true), [], 1);
    const child2 = makeNode(makeTask(3n, false), [], 1);
    const child3 = makeNode(makeTask(4n, true), [], 1);
    const parent = makeNode(makeTask(1n, false), [child1, child2, child3]);
    const result = filterTree([parent], false);
    expect(result).toHaveLength(1);
    expect(result[0].task.id).toBe(1n);
    expect(result[0].children).toEqual([child2]);
  });

  it("prunes a fully-completed subtree", () => {
    const child = makeNode(makeTask(2n, true), [], 1);
    const parent = makeNode(makeTask(1n, true), [child]);
    expect(filterTree([parent], false)).toEqual([]);
  });

  it("keeps a completed ancestor that has an incomplete descendant", () => {
    const grandchild = makeNode(makeTask(3n, false), [], 2);
    const child = makeNode(makeTask(2n, true), [grandchild], 1);
    const parent = makeNode(makeTask(1n, true), [child]);
    const result = filterTree([parent], false);
    expect(result).toHaveLength(1);
    expect(result[0].task.id).toBe(1n);
    expect(result[0].children[0].task.id).toBe(2n);
    expect(result[0].children[0].children[0].task.id).toBe(3n);
  });

  it("does not mutate input nodes", () => {
    const child = makeNode(makeTask(2n, true), [], 1);
    const parent = makeNode(makeTask(1n, false), [child]);
    const original = { ...parent, children: [...parent.children] };
    filterTree([parent], false);
    expect(parent.children).toEqual(original.children);
  });

  it("filters a completed grandchild even when its parent (and grandparent) are incomplete", () => {
    const grandchild = makeNode(makeTask(3n, true), [], 2);
    const child = makeNode(makeTask(2n, false), [grandchild], 1);
    const parent = makeNode(makeTask(1n, false), [child]);
    const result = filterTree([parent], false);
    expect(result).toHaveLength(1);
    expect(result[0].task.id).toBe(1n);
    expect(result[0].children).toHaveLength(1);
    expect(result[0].children[0].task.id).toBe(2n);
    expect(result[0].children[0].children).toHaveLength(0);
  });

  it("preserves depth values", () => {
    const child = makeNode(makeTask(2n, false), [], 3);
    const parent = makeNode(makeTask(1n, false), [child], 2);
    const result = filterTree([parent], false);
    expect(result[0].depth).toBe(2);
    expect(result[0].children[0].depth).toBe(3);
  });
});

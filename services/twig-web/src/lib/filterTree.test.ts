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

// --- Snooze filtering tests ---

function makeSnoozedTask(id: bigint, snoozeSeconds: bigint, parentId?: bigint): Task {
  return {
    id,
    name: `Task ${id}`,
    description: "",
    parentId,
    due: undefined,
    completedAt: undefined,
    estimate: 0,
    snoozeUntil: { seconds: snoozeSeconds, nanos: 0 },
  } as unknown as Task;
}

// Reference date: local noon on 2026-06-06 (getDate()=6 in any timezone)
const TODAY = new Date(2026, 5, 6, 12, 0, 0);
// 2026-06-07T00:00:00Z — tomorrow UTC = 1780790400 seconds
const TOMORROW_SECONDS = 1780790400n;
// 2026-06-06T00:00:00Z — today UTC = 1780704000 seconds
const TODAY_SECONDS = 1780704000n;
// 2026-06-05T00:00:00Z — yesterday UTC = 1780617600 seconds
const YESTERDAY_SECONDS = 1780617600n;

describe("filterTree snooze", () => {
  it("hides a future-snoozed task in default view", () => {
    const snoozed = makeNode(makeSnoozedTask(1n, TOMORROW_SECONDS));
    const active = makeNode(makeTask(2n, false));
    const result = filterTree([snoozed, active], false, TODAY);
    expect(result).toHaveLength(1);
    expect(result[0].task.id).toBe(2n);
  });

  it("shows snoozed tasks when showCompleted is true (show-all)", () => {
    const snoozed = makeNode(makeSnoozedTask(1n, TOMORROW_SECONDS));
    const nodes = [snoozed];
    expect(filterTree(nodes, true, TODAY)).toEqual(nodes);
  });

  it("keeps task snoozed for today (not strictly after)", () => {
    const todaySnoozed = makeNode(makeSnoozedTask(1n, TODAY_SECONDS));
    const result = filterTree([todaySnoozed], false, TODAY);
    expect(result).toHaveLength(1);
  });

  it("keeps task snoozed for yesterday", () => {
    const yesterday = makeNode(makeSnoozedTask(1n, YESTERDAY_SECONDS));
    const result = filterTree([yesterday], false, TODAY);
    expect(result).toHaveLength(1);
  });

  it("hides entire subtree when parent is snoozed", () => {
    const child = makeNode(makeTask(2n, false), [], 1);
    const parent = makeNode(makeSnoozedTask(1n, TOMORROW_SECONDS), [child]);
    const result = filterTree([parent], false, TODAY);
    expect(result).toHaveLength(0);
  });

  it("hides snoozed child but keeps incomplete parent", () => {
    const snoozedChild = makeNode(makeSnoozedTask(2n, TOMORROW_SECONDS), [], 1);
    const activeChild = makeNode(makeTask(3n, false), [], 1);
    const parent = makeNode(makeTask(1n, false), [snoozedChild, activeChild]);
    const result = filterTree([parent], false, TODAY);
    expect(result).toHaveLength(1);
    expect(result[0].children).toHaveLength(1);
    expect(result[0].children[0].task.id).toBe(3n);
  });
});

import { describe, it, expect } from "vitest";
import { effectiveGoal } from "./effectiveGoal";
import type { Task } from "../gen/task/v1/task_pb";

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 1n,
    name: "Task",
    description: "",
    parentId: undefined,
    goalId: undefined,
    completedAt: undefined,
    estimate: 0,
    $typeName: "task.v1.Task",
    ...overrides,
  } as unknown as Task;
}

describe("effectiveGoal", () => {
  it("resolves a task with its own goalId as direct", () => {
    const tasks = [makeTask({ id: 1n, goalId: 10n })];
    const result = effectiveGoal(1n, tasks);
    expect(result).toEqual({ goalId: 10n, kind: "direct" });
  });

  it("resolves a task whose nearest goal-bearing ancestor has one as inherited", () => {
    const tasks = [
      makeTask({ id: 1n, goalId: 10n }),
      makeTask({ id: 2n, parentId: 1n }),
    ];
    const result = effectiveGoal(2n, tasks);
    expect(result).toEqual({ goalId: 10n, kind: "inherited" });
  });

  it("the nearest ancestor wins when several ancestors carry goals", () => {
    const tasks = [
      makeTask({ id: 1n, goalId: 10n }),
      makeTask({ id: 2n, parentId: 1n, goalId: 20n }),
      makeTask({ id: 3n, parentId: 2n }),
    ];
    const result = effectiveGoal(3n, tasks);
    expect(result).toEqual({ goalId: 20n, kind: "inherited" });
  });

  it("returns none for a task with no goal anywhere in its chain", () => {
    const tasks = [
      makeTask({ id: 1n }),
      makeTask({ id: 2n, parentId: 1n }),
    ];
    expect(effectiveGoal(2n, tasks)).toBeUndefined();
  });

  it("returns none for a root task with no goal", () => {
    const tasks = [makeTask({ id: 1n })];
    expect(effectiveGoal(1n, tasks)).toBeUndefined();
  });

  it("terminates without looping on a missing/broken parentId link", () => {
    const tasks = [makeTask({ id: 2n, parentId: 999n })];
    expect(effectiveGoal(2n, tasks)).toBeUndefined();
  });

  it("terminates without looping on a cyclic parentId chain", () => {
    const tasks = [
      makeTask({ id: 1n, parentId: 2n }),
      makeTask({ id: 2n, parentId: 1n }),
    ];
    expect(effectiveGoal(1n, tasks)).toBeUndefined();
  });
});

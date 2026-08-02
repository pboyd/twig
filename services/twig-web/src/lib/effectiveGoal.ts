import type { Task } from "../gen/task/v1/task_pb";

export interface EffectiveGoal {
  goalId: bigint;
  kind: "direct" | "inherited";
}

export function effectiveGoal(taskId: bigint, tasks: Task[]): EffectiveGoal | undefined {
  const byId = new Map(tasks.map((t) => [t.id, t]));
  const visited = new Set<bigint>();

  let current = byId.get(taskId);
  let isSelf = true;
  while (current && !visited.has(current.id)) {
    visited.add(current.id);
    if (current.goalId !== undefined) {
      return { goalId: current.goalId, kind: isSelf ? "direct" : "inherited" };
    }
    isSelf = false;
    current = current.parentId !== undefined ? byId.get(current.parentId) : undefined;
  }

  return undefined;
}

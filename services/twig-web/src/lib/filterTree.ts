import type { Task } from "../gen/task/v1/task_pb";
import type { TaskNode } from "./tree";

function isSnoozed(task: Task, refDate: Date): boolean {
  if (!task.snoozeUntil) return false;
  const snooze = new Date(Number(task.snoozeUntil.seconds) * 1000);
  // Compare UTC calendar date of snoozeUntil to local calendar date of refDate.
  if (snooze.getUTCFullYear() !== refDate.getFullYear())
    return snooze.getUTCFullYear() > refDate.getFullYear();
  if (snooze.getUTCMonth() !== refDate.getMonth())
    return snooze.getUTCMonth() > refDate.getMonth();
  return snooze.getUTCDate() > refDate.getDate();
}

export function filterTree(nodes: TaskNode[], showCompleted: boolean, refDate?: Date): TaskNode[] {
  if (showCompleted) return nodes;

  const today = refDate ?? new Date();

  function keep(node: TaskNode): TaskNode | null {
    // Snoozed nodes hide their entire subtree — no children are promoted.
    if (isSnoozed(node.task, today)) return null;

    const filteredChildren = node.children.flatMap((c) => {
      const result = keep(c);
      return result ? [result] : [];
    });

    const hasKeptDescendant = filteredChildren.length > 0;
    const isIncomplete = node.task.completedAt === undefined;

    if (!isIncomplete && !hasKeptDescendant) return null;

    const childrenUnchanged =
      filteredChildren.length === node.children.length &&
      filteredChildren.every((c, i) => c === node.children[i]);

    return childrenUnchanged ? node : { ...node, children: filteredChildren };
  }

  return nodes.flatMap((n) => {
    const result = keep(n);
    return result ? [result] : [];
  });
}

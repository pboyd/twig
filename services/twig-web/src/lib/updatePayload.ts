import type { Task } from "../gen/task/v1/task_pb";

// UpdateTask is full-replace on every editable field including parent_id.
// Always pass the task's current parentId, even on a plain rename.
// Omitting parentId is interpreted as a promotion-to-root: the server
// writes any goal the task was inheriting from a parent (see
// specs/072-move-task-goal-rules/contracts/update-task-goal-behavior.md).
export function buildUpdatePayload(
  task: Task,
  editedName: string,
  editedDescription: string
) {
  return {
    id: task.id,
    name: editedName,
    description: editedDescription,
    due: task.due,
    parentId: task.parentId,
    snoozeUntil: task.snoozeUntil,
  };
}

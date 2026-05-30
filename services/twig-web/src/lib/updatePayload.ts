import type { Task } from "../gen/task/v1/task_pb";

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
  };
}

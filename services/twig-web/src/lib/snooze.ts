import type { Task } from "../gen/task/v1/task_pb";

// Compare UTC calendar date of snoozeUntil to local calendar date of refDate.
export function isSnoozed(task: Task | undefined, refDate?: Date): boolean {
  if (!task?.snoozeUntil) return false;
  const today = refDate ?? new Date();
  const snooze = new Date(Number(task.snoozeUntil.seconds) * 1000);
  if (snooze.getUTCFullYear() !== today.getFullYear())
    return snooze.getUTCFullYear() > today.getFullYear();
  if (snooze.getUTCMonth() !== today.getMonth())
    return snooze.getUTCMonth() > today.getMonth();
  return snooze.getUTCDate() > today.getDate();
}

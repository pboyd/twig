import { GoalState } from "../gen/goal/v1/goal_pb";
import type { Goal } from "../gen/goal/v1/goal_pb";
import { Markdown } from "./Markdown";
import { formatDueDate } from "../lib/formatTimestamp";

interface GoalListItemProps {
  goal: Goal;
  onOpen: (id: bigint) => void;
}

const STATE_COLORS: Partial<Record<GoalState, string>> = {
  [GoalState.IN_PROGRESS]: "border-l-green-500",
  [GoalState.INCUBATING]: "border-l-blue-400",
  [GoalState.COMPLETED]: "border-l-gray-400",
  [GoalState.ARCHIVED]: "border-l-gray-400",
};

const STATE_LABELS: Partial<Record<GoalState, string>> = {
  [GoalState.IN_PROGRESS]: "In Progress",
  [GoalState.INCUBATING]: "Incubating",
  [GoalState.COMPLETED]: "Completed",
  [GoalState.ARCHIVED]: "Archived",
};

export function GoalListItem({ goal, onOpen }: GoalListItemProps) {

  return (
    <button
      type="button"
      onClick={() => onOpen(goal.id)}
      className={[
        "w-full text-left rounded-lg border border-gray-200 border-l-4 bg-white p-3 shadow-sm transition-shadow hover:shadow-md dark:border-gray-700 dark:bg-gray-800",
        STATE_COLORS[goal.state] || "border-l-gray-300",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
            <Markdown mode="inline">{goal.name}</Markdown>
          </span>
          {goal.due && (
            <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              Due {formatDueDate(goal.due)}
            </p>
          )}
        </div>
        <span className="shrink-0 rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">
          {STATE_LABELS[goal.state] ?? String(goal.state)}
        </span>
      </div>
    </button>
  );
}

import { messages } from "../theme/messages";

interface EmptyStateProps {
  onAddTask?: () => void;
}

export function EmptyState({ onAddTask }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center gap-4">
      <p className="text-sm text-gray-500 dark:text-gray-400">{messages.emptyTaskList}</p>
      {onAddTask && (
        <button
          onClick={onAddTask}
          className="text-sm text-blue-600 dark:text-blue-400 hover:underline focus:outline-none"
        >
          Add the first task →
        </button>
      )}
    </div>
  );
}

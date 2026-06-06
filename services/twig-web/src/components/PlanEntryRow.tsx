import { Link } from "react-router";
import type { ResolvedEntry } from "../lib/planView";

interface PlanEntryRowProps {
  entry: ResolvedEntry;
}

export function PlanEntryRow({ entry }: PlanEntryRowProps) {
  const nameContent = (
    <span
      className={entry.completed ? "line-through text-gray-400 dark:text-gray-500" : ""}
    >
      {entry.displayName}
    </span>
  );

  return (
    <div className="flex flex-col gap-0.5 py-2 px-4 border-b border-gray-100 dark:border-gray-800/60 last:border-0">
      {entry.timed && entry.timeLabel && (
        <span className="text-xs text-gray-500 dark:text-gray-400 font-mono tabular-nums">
          {entry.timeLabel}
        </span>
      )}
      <div className="flex items-center gap-2">
        {entry.completed && (
          <span
            data-testid="completed-marker"
            aria-label="Completed"
            className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-green-500 text-white text-xs shrink-0"
          >
            ✓
          </span>
        )}
        {entry.kind === "task" && entry.taskId !== undefined ? (
          <Link
            to={`/tasks/${entry.taskId}`}
            className="text-sm font-medium text-blue-700 dark:text-blue-400 hover:underline"
          >
            {nameContent}
          </Link>
        ) : (
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300 italic">
            {nameContent}
          </span>
        )}
      </div>
    </div>
  );
}

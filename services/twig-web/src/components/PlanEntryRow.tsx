import { Link } from "react-router";
import type { ResolvedEntry } from "../lib/planView";
import { Markdown } from "./Markdown";
import { CompletionToggle } from "./CompletionToggle";

interface PlanEntryRowProps {
  entry: ResolvedEntry;
  onToggleComplete?: (entry: ResolvedEntry) => void;
  onRemove?: (entry: ResolvedEntry) => void;
  completing?: boolean;
  removing?: boolean;
  actionError?: string;
}

export function PlanEntryRow({
  entry,
  onToggleComplete,
  onRemove,
  completing,
  removing,
  actionError,
}: PlanEntryRowProps) {
  const nameContent = (
    <span
      className={entry.completed ? "line-through text-gray-400 dark:text-gray-500" : ""}
    >
      <Markdown mode="inline">{entry.displayName}</Markdown>
    </span>
  );

  return (
    <div className="flex flex-col gap-0.5 py-2 px-4 border-b border-gray-100 dark:border-gray-800/60 last:border-0">
      <div className="flex items-center gap-2">
        {entry.kind === "task" && entry.taskId !== undefined ? (
          <>
            <CompletionToggle
              completed={entry.completed}
              disabled={completing}
              onToggle={() => onToggleComplete?.(entry)}
            />
            <Link
              to={`/tasks/${entry.taskId}`}
              className="flex-1 text-sm font-medium text-blue-700 dark:text-blue-400 hover:underline"
            >
              {nameContent}
            </Link>
          </>
        ) : (
          <span className="flex-1 text-sm font-medium text-gray-700 dark:text-gray-300 italic">
            {nameContent}
          </span>
        )}

        {onRemove !== undefined && (
          <button
            aria-label="Remove entry"
            title="Remove from plan"
            disabled={removing}
            data-testid="remove-btn"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
            onClick={() => onRemove(entry)}
          >
            <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path
                fillRule="evenodd"
                d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        )}
      </div>

      {actionError && (
        <p className="text-sm text-amber-700 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400 px-1 py-0.5 rounded">
          {actionError}
        </p>
      )}
    </div>
  );
}

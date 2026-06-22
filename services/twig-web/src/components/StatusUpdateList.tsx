import type { StatusUpdate } from "../gen/goal/v1/goal_pb";
import { Markdown } from "./Markdown";
import { formatTimestamp } from "../lib/formatTimestamp";

interface StatusUpdateListProps {
  updates: StatusUpdate[];
  onEdit?: (update: StatusUpdate) => void;
  onDelete?: (id: bigint) => void;
}

export function StatusUpdateList({ updates, onEdit, onDelete }: StatusUpdateListProps) {
  if (updates.length === 0) return null;

  return (
    <div className="space-y-3">
      {updates.map((update) => (
        <div
          key={String(update.id)}
          className="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800"
        >
          <div className="mb-1 flex items-center justify-between gap-2">
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatTimestamp(update.createdAt)}
            </span>
            {(onEdit || onDelete) && (
              <div className="flex gap-1">
                {onEdit && (
                  <button
                    type="button"
                    onClick={() => onEdit(update)}
                    className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                  >
                    Edit
                  </button>
                )}
                {onDelete && (
                  <button
                    type="button"
                    onClick={() => onDelete(update.id)}
                    className="text-xs text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                  >
                    Delete
                  </button>
                )}
              </div>
            )}
          </div>
          <div className="prose-sm max-w-none overflow-x-auto text-sm text-gray-700 dark:text-gray-300">
            <Markdown mode="block">{update.body}</Markdown>
          </div>
        </div>
      ))}
    </div>
  );
}

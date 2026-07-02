import { useEffect, useState } from "react";
import { Link } from "react-router";
import type { ResolvedEntry } from "../lib/planView";
import {
  SLOT_MINUTES,
  SLOT_PX,
  snapDown15,
  snapUp15,
  computeWindow,
  slotIndex,
  slotCount,
  hourLabels,
  isToday as isTodayString,
} from "../lib/planView";
import { Markdown } from "./Markdown";
import { CompletionToggle } from "./CompletionToggle";

interface PlanTimelineProps {
  entries: ResolvedEntry[];
  day: string;
  onToggleComplete: (entry: ResolvedEntry) => void;
  onRemove: (entry: ResolvedEntry) => void;
  pendingIds: Set<number>;
  entryErrors: Map<number, string>;
}

export function PlanTimeline({
  entries,
  day,
  onToggleComplete,
  onRemove,
  pendingIds,
  entryErrors,
}: PlanTimelineProps) {
  const window = computeWindow(entries);
  const totalSlots = slotCount(window);
  const labels = hourLabels(window);

  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const interval = setInterval(() => setNow(new Date()), 60_000);
    return () => clearInterval(interval);
  }, []);

  const isToday = isTodayString(day);
  const nowMinute = now.getHours() * 60 + now.getMinutes();
  const showNowIndicator = isToday && nowMinute >= window.startMinute && nowMinute < window.endMinute;
  const nowOffset = showNowIndicator ? ((nowMinute - window.startMinute) / SLOT_MINUTES) * SLOT_PX : 0;

  return (
    <div className="relative">
      <div
        className="grid"
        style={{
          gridTemplateColumns: "60px 1fr",
          gridTemplateRows: `repeat(${totalSlots}, ${SLOT_PX}px)`,
        }}
      >
        {labels.map(({ minute, label }) => {
          const idx = slotIndex(window, minute);
          return (
            <div
              key={minute}
              className="col-start-1 row-start-1 flex items-start justify-end pr-2 pt-0 text-xs text-gray-500 dark:text-gray-400 font-mono tabular-nums select-none"
              style={{ gridRow: `${idx + 1} / span 1`, height: 0, overflow: "visible" }}
            >
              {label}
            </div>
          );
        })}

        {labels.map(({ minute }) => {
          const idx = slotIndex(window, minute);
          return (
            <div
              key={`ruler-${minute}`}
              className="col-start-2 row-start-1 border-t border-gray-200 dark:border-gray-700 pointer-events-none"
              style={{ gridRow: `${idx + 1} / span 1`, height: 0, overflow: "visible" }}
            />
          );
        })}

        {entries.map((entry) => {
          if (entry.startMinute === undefined || entry.endMinute === undefined) return null;
          const sIdx = slotIndex(window, snapDown15(entry.startMinute));
          const eIdx = slotIndex(window, snapUp15(entry.endMinute));
          const span = Math.max(eIdx - sIdx, 1);

          return (
            <div
              key={entry.id}
              className={[
                "col-start-2 flex items-center gap-1 px-2 border-t border-gray-100 dark:border-gray-800/40 relative",
                entry.completed ? "bg-gray-50 dark:bg-gray-800/20" : "",
              ].join(" ")}
              style={{ gridRow: `${sIdx + 1} / span ${span}` }}
            >
              {entry.kind === "task" && entry.taskId !== undefined ? (
                <>
                  <CompletionToggle
                    completed={entry.completed}
                    disabled={pendingIds.has(entry.id)}
                    onToggle={() => onToggleComplete(entry)}
                  />
                  <Link
                    to={`/tasks/${entry.taskId}`}
                    className={[
                      "flex-1 text-sm font-medium truncate",
                      entry.completed
                        ? "line-through text-gray-400 dark:text-gray-500"
                        : "text-blue-700 dark:text-blue-400 hover:underline",
                    ].join(" ")}
                  >
                    <Markdown mode="inline">{entry.displayName}</Markdown>
                  </Link>
                </>
              ) : (
                <span
                  className={[
                    "flex-1 text-sm font-medium truncate italic",
                    entry.completed
                      ? "line-through text-gray-400 dark:text-gray-500"
                      : "text-gray-700 dark:text-gray-300",
                  ].join(" ")}
                >
                  <Markdown mode="inline">{entry.displayName}</Markdown>
                </span>
              )}

              {onRemove !== undefined && (
                <button
                  aria-label="Remove entry"
                  title="Remove from plan"
                  disabled={pendingIds.has(entry.id)}
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

              {entryErrors.has(entry.id) && (
                <p className="absolute bottom-0 left-0 right-0 text-xs text-amber-700 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400 px-1 py-0.5 rounded translate-y-full">
                  {entryErrors.get(entry.id)}
                </p>
              )}
            </div>
          );
        })}
      </div>

      {showNowIndicator && (
        <div
          data-testid="now-indicator"
          className="absolute left-0 right-0 pointer-events-none z-10"
          style={{ top: `${nowOffset}px` }}
        >
          <div className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full bg-blue-500 dark:bg-blue-400 shrink-0" />
            <span className="h-px flex-1 bg-blue-500 dark:bg-blue-400" />
          </div>
        </div>
      )}
    </div>
  );
}

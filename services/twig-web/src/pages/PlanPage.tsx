import { useState } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { listPlanEntries } from "../gen/plan/v1/plan-PlanService_connectquery";
import { listTasks } from "../gen/task/v1/task-TaskService_connectquery";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { PlanEntryRow } from "../components/PlanEntryRow";
import { messages } from "../theme/messages";
import {
  todayString,
  addDays,
  isToday,
  formatDayLabel,
  resolveEntries,
  groupPlan,
} from "../lib/planView";
import { Button } from "../components/Button";

export default function PlanPage() {
  const [day, setDay] = useState(todayString);

  const planQuery = useQuery(listPlanEntries, { day });
  const tasksQuery = useQuery(listTasks, {});

  const taskNameById = new Map<bigint, string>(
    (tasksQuery.data?.tasks ?? []).map((t) => [t.id, t.name])
  );

  const entries = planQuery.data?.entries ?? [];
  const resolved = resolveEntries(entries, taskNameById);
  const grouped = groupPlan(resolved);

  const isLoading = planQuery.isLoading;
  const isError = planQuery.isError;

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 pb-8">
        {/* Day navigation */}
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              aria-label="Previous day"
              onClick={() => setDay((d) => addDays(d, -1))}
              className="text-sm px-3"
            >
              ←
            </Button>
            <Button
              variant="secondary"
              aria-label="Next day"
              onClick={() => setDay((d) => addDays(d, 1))}
              className="text-sm px-3"
            >
              →
            </Button>
          </div>
          <h1 className="text-base font-semibold text-gray-900 dark:text-gray-100">
            {formatDayLabel(day)}
          </h1>
          <Button
            variant="secondary"
            onClick={() => setDay(todayString())}
            disabled={isToday(day)}
            className="text-sm px-3"
          >
            Today
          </Button>
        </div>

        {isLoading && (
          <div className="flex justify-center py-12" role="status" aria-label="Loading">
            <Spinner size="lg" />
          </div>
        )}

        {isError && (
          <div className="mx-4">
            <ErrorBanner
              message={messages.planError}
              onRetry={() => planQuery.refetch()}
            />
          </div>
        )}

        {!isLoading && !isError && grouped.isEmpty && (
          <div className="mx-4 mt-8 text-center text-gray-500 dark:text-gray-400">
            <p>{messages.planEmpty}</p>
          </div>
        )}

        {!isLoading && !isError && !grouped.isEmpty && (
          <>
            {grouped.timed.length > 0 && (
              <section data-testid="timed-section">
                <ul className="list-none p-0 m-0 border-t border-gray-100 dark:border-gray-800/60">
                  {grouped.timed.map((entry) => (
                    <li key={entry.id}>
                      <PlanEntryRow entry={entry} />
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {grouped.untimed.length > 0 && (
              <section data-testid="untimed-section" className="mt-4">
                <h2 className="px-4 pb-1 text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
                  Untimed
                </h2>
                <ul className="list-none p-0 m-0 border-t border-gray-100 dark:border-gray-800/60">
                  {grouped.untimed.map((entry) => (
                    <li key={entry.id}>
                      <PlanEntryRow entry={entry} />
                    </li>
                  ))}
                </ul>
              </section>
            )}
          </>
        )}
      </main>
    </div>
  );
}

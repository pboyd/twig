import { useState } from "react";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { ConnectError, Code } from "@connectrpc/connect";
import { listPlanEntries, removePlanEntry } from "../gen/plan/v1/plan-PlanService_connectquery";
import { listTasks, completeTask } from "../gen/task/v1/task-TaskService_connectquery";
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
import type { ResolvedEntry } from "../lib/planView";
import { Button } from "../components/Button";
import { useToast } from "../context/ToastProvider";

export default function PlanPage() {
  const [day, setDay] = useState(todayString);
  const [entryErrors, setEntryErrors] = useState<Map<number, string>>(new Map());
  const [completingIds, setCompletingIds] = useState<Set<number>>(new Set());
  const [removingIds, setRemovingIds] = useState<Set<number>>(new Set());
  const { show } = useToast();
  const queryClient = useQueryClient();

  const planQuery = useQuery(listPlanEntries, { day });
  const tasksQuery = useQuery(listTasks, {});

  const { mutateAsync: doComplete } = useMutation(completeTask);
  const { mutateAsync: doRemove } = useMutation(removePlanEntry);

  const planQueryKey = createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" });
  const tasksQueryKey = createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" });

  const taskNameById = new Map<bigint, string>(
    (tasksQuery.data?.tasks ?? []).map((t) => [t.id, t.name])
  );

  const entries = planQuery.data?.entries ?? [];
  const resolved = resolveEntries(entries, taskNameById);
  const grouped = groupPlan(resolved);

  const isLoading = planQuery.isLoading;
  const isError = planQuery.isError;

  function clearEntryError(id: number) {
    setEntryErrors((prev) => {
      const next = new Map(prev);
      next.delete(id);
      return next;
    });
  }

  async function handleComplete(entry: ResolvedEntry) {
    if (entry.taskId === undefined) return;
    clearEntryError(entry.id);
    setCompletingIds((prev) => new Set(prev).add(entry.id));
    try {
      await doComplete({ id: entry.taskId });
      await queryClient.invalidateQueries({ queryKey: planQueryKey });
      await queryClient.invalidateQueries({ queryKey: tasksQueryKey });
    } catch (err) {
      const msg =
        err instanceof ConnectError && err.code === Code.FailedPrecondition
          ? messages.completeBlockedBySubtasks
          : messages.connectivityError;
      setEntryErrors((prev) => new Map(prev).set(entry.id, msg));
    } finally {
      setCompletingIds((prev) => {
        const next = new Set(prev);
        next.delete(entry.id);
        return next;
      });
    }
  }

  async function handleRemove(entry: ResolvedEntry) {
    clearEntryError(entry.id);
    setRemovingIds((prev) => new Set(prev).add(entry.id));
    try {
      await doRemove({ day, id: entry.id });
      await queryClient.invalidateQueries({ queryKey: planQueryKey });
      show(messages.entryRemoved, "success");
    } catch {
      setEntryErrors((prev) => new Map(prev).set(entry.id, messages.connectivityError));
    } finally {
      setRemovingIds((prev) => {
        const next = new Set(prev);
        next.delete(entry.id);
        return next;
      });
    }
  }

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
                      <PlanEntryRow
                        entry={entry}
                        onComplete={handleComplete}
                        onRemove={handleRemove}
                        completing={completingIds.has(entry.id)}
                        removing={removingIds.has(entry.id)}
                        actionError={entryErrors.get(entry.id)}
                      />
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
                      <PlanEntryRow
                        entry={entry}
                        onComplete={handleComplete}
                        onRemove={handleRemove}
                        completing={completingIds.has(entry.id)}
                        removing={removingIds.has(entry.id)}
                        actionError={entryErrors.get(entry.id)}
                      />
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

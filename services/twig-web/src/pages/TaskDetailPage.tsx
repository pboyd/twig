import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getTask,
  listTasks,
  updateTask,
  deleteTask,
  completeTask,
  uncompleteTask,
  setTaskGoal,
} from "../gen/task/v1/task-TaskService_connectquery";
import { TaskService } from "../gen/task/v1/task_pb";
import { listGoals } from "../gen/goal/v1/goal-GoalService_connectquery";
import { createClient, ConnectError, Code } from "@connectrpc/connect";
import { transport } from "../lib/transport";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { Button } from "../components/Button";
import { TaskForm } from "../components/TaskForm";
import { messages } from "../theme/messages";
import { buildUpdatePayload } from "../lib/updatePayload";
import { Markdown } from "../components/Markdown";
import { useToast } from "../context/ToastProvider";
import { formatDueDate } from "../lib/formatTimestamp";
import { resolveDayEdit, timestampToIsoDay } from "../lib/dateFields";
import { effectiveGoal } from "../lib/effectiveGoal";
import { goalGroups } from "../lib/goalGroups";
import { isSnoozed } from "../lib/snooze";

export default function TaskDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const toast = useToast();
  const taskId = id ? BigInt(id) : undefined;

  const [isEditing, setIsEditing] = useState(false);
  const [toggleError, setToggleError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [goalError, setGoalError] = useState<string | null>(null);

  const { data, isLoading, isError, error } = useQuery(
    getTask,
    taskId !== undefined ? { id: taskId } : undefined,
    { enabled: taskId !== undefined }
  );

  const { data: allTasksData } = useQuery(listTasks, {});
  const subTasks = (allTasksData?.tasks ?? []).filter(
    (t) => t.parentId !== undefined && t.parentId === taskId
  );

  const { data: goalsData } = useQuery(listGoals, {});

  const { mutateAsync: doUpdate, isPending: isUpdating } = useMutation(updateTask);
  const { mutateAsync: doComplete, isPending: isCompleting } = useMutation(completeTask);
  const { mutateAsync: doUncomplete, isPending: isUncompleting } = useMutation(uncompleteTask);
  const { mutateAsync: doDelete, isPending: isDeleting } = useMutation(deleteTask);
  const { mutateAsync: doSetGoal } = useMutation(setTaskGoal);

  const task = data?.task;
  const is404 =
    isError &&
    error instanceof ConnectError &&
    error.code === Code.NotFound;

  async function handleSaveEdit(
    editedName: string,
    editedDesc: string,
    _planDay: string | undefined,
    editedDue?: string,
    editedSnooze?: string
  ) {
    if (!task) return;
    // UpdateTask is full-replace on parentId (see updatePayload.ts). The
    // cached `task` can be stale if the parent changed elsewhere while this
    // page was open, so refetch immediately before building the payload —
    // otherwise a plain rename can replay a stale parentId and the server
    // reads it as a promotion, rewriting the task's goal link.
    let current = task;
    try {
      const fresh = await createClient(TaskService, transport).getTask({ id: taskId! });
      if (fresh.task) current = fresh.task;
    } catch {
      setToggleError(messages.connectivityError);
      return;
    }
    const payload = buildUpdatePayload(
      current,
      editedName,
      editedDesc,
      resolveDayEdit(current.due, editedDue),
      resolveDayEdit(current.snoozeUntil, editedSnooze)
    );
    try {
      await doUpdate(payload);
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: getTask, input: { id: taskId }, cardinality: "finite" }),
      });
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
      });
      setIsEditing(false);
    } catch {
      setToggleError(messages.saveError);
    }
  }

  async function handleToggleComplete() {
    if (!task) return;
    setToggleError(null);
    try {
      if (task.completedAt) {
        await doUncomplete({ id: task.id });
      } else {
        await doComplete({ id: task.id });
      }
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: getTask, input: { id: taskId }, cardinality: "finite" }),
      });
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
      });
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
        setToggleError(
          task.completedAt
            ? messages.reopenBlockedByParent
            : messages.completeBlockedBySubtasks
        );
      } else {
        setToggleError(messages.connectivityError);
      }
    }
  }

  async function handleConfirmDelete() {
    if (!task) return;
    setDeleteError(null);
    try {
      await doDelete({ id: task.id });
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
      });
      toast.show(messages.taskDeleted);
      navigate("/tasks");
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.NotFound) {
        await queryClient.invalidateQueries({
          queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
        });
        toast.show(messages.taskAlreadyGone);
        navigate("/tasks");
      } else {
        setDeleteError(messages.connectivityError);
      }
    }
  }

  async function handleSetGoal(goalId: bigint | undefined) {
    if (!task) return;
    setGoalError(null);
    try {
      if (goalId !== undefined) {
        await doSetGoal({ taskId: task.id, goalId });
      } else {
        await doSetGoal({ taskId: task.id });
      }
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: getTask, input: { id: taskId }, cardinality: "finite" }),
      });
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
      });
      toast.show(goalId !== undefined ? messages.goalLinked : messages.goalUnlinked);
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
        setGoalError(messages.goalLinkBlocked);
      } else if (err instanceof ConnectError && err.code === Code.NotFound) {
        setGoalError(messages.taskNotFound);
        await queryClient.invalidateQueries({
          queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
        });
      } else {
        setGoalError(messages.connectivityError);
      }
    }
  }

  const isToggling = isCompleting || isUncompleting;

  const snoozed = isSnoozed(task);

  const allGoals = goalsData?.goals ?? [];
  const goalNameById = new Map(allGoals.map((g) => [g.id, g.name]));
  const selectableGoals = goalGroups(allGoals, false).flatMap((g) => g.goals);
  const goal = task ? effectiveGoal(task.id, allTasksData?.tasks ?? []) : undefined;
  // The direct-goal select is controlled by goal.goalId; if that goal is
  // completed/archived it's excluded from selectableGoals (goalGroups hides
  // those states), which would leave the select's value with no matching
  // option. Make sure the currently-linked goal is always present.
  const linkedGoal = goal?.kind === "direct" ? allGoals.find((g) => g.id === goal.goalId) : undefined;
  const directGoalOptions =
    linkedGoal && !selectableGoals.some((g) => g.id === linkedGoal.id)
      ? [...selectableGoals, linkedGoal]
      : selectableGoals;

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-4 py-4">
        <Button
          variant="secondary"
          onClick={() => navigate("/tasks")}
          className="mb-4 text-sm px-3"
        >
          ← Back
        </Button>

        {isLoading && (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        )}

        {isError && (
          <ErrorBanner
            message={is404 ? messages.taskNotFound : (error?.message ?? messages.connectivityError)}
          />
        )}

        {toggleError && (
          <div className="mb-4">
            <ErrorBanner message={toggleError} />
          </div>
        )}

        {deleteError && (
          <div className="mb-4">
            <ErrorBanner message={deleteError} />
          </div>
        )}

        {goalError && (
          <div className="mb-4">
            <ErrorBanner message={goalError} />
          </div>
        )}

        {task && !isEditing && (
          <div className="space-y-4">
            <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
              <div className="flex items-start gap-3">
                <button
                  aria-label={task.completedAt ? "Mark incomplete" : "Mark complete"}
                  disabled={isToggling}
                  onClick={handleToggleComplete}
                  className={[
                    "mt-0.5 flex h-11 w-11 shrink-0 -ml-3 items-center justify-center rounded",
                    isToggling ? "opacity-50" : "cursor-pointer",
                  ].join(" ")}
                >
                  <span
                    className={[
                      "flex h-5 w-5 items-center justify-center rounded-full border-2",
                      task.completedAt
                        ? "border-green-500 bg-green-500 text-white"
                        : "border-gray-400 dark:border-gray-500",
                    ].join(" ")}
                  >
                    {task.completedAt && (
                      <svg className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                        <path
                          fillRule="evenodd"
                          d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
                          clipRule="evenodd"
                        />
                      </svg>
                    )}
                  </span>
                </button>
                <div className="flex-1 min-w-0">
                  <h1
                    className={[
                      "text-xl font-semibold leading-snug",
                      task.completedAt
                        ? "line-through text-gray-400 dark:text-gray-500"
                        : "text-gray-900 dark:text-gray-100",
                    ].join(" ")}
                  >
                    <Markdown mode="inline">{task.name}</Markdown>
                  </h1>
                  {task.description && (
                    <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                      <Markdown mode="block">{task.description}</Markdown>
                    </div>
                  )}
                  <p className="mt-3 text-xs text-gray-400 dark:text-gray-500">
                    {task.completedAt ? "Completed" : "Incomplete"}
                  </p>
                  <p className="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {messages.dueLabel}: {task.due ? formatDueDate(task.due) : messages.noDueDate}
                  </p>
                  {snoozed && (
                    <p className="mt-1 text-xs text-gray-400 dark:text-gray-500">
                      {messages.snoozedUntil(formatDueDate(task.snoozeUntil))} 💤
                    </p>
                  )}
                </div>
              </div>
              {confirmingDelete ? (
                <div className="mt-4 rounded-md bg-red-50 p-3 dark:bg-red-900/20">
                  <p className="text-sm text-red-800 dark:text-red-300">
                    {subTasks.length > 0
                      ? messages.deleteTaskConfirmWithSubtasks
                      : messages.deleteTaskConfirm}
                  </p>
                  <div className="mt-3 flex justify-end gap-2">
                    <Button
                      variant="secondary"
                      onClick={() => setConfirmingDelete(false)}
                      className="text-sm px-3"
                    >
                      {messages.cancelButton}
                    </Button>
                    <Button
                      variant="danger"
                      loading={isDeleting}
                      onClick={handleConfirmDelete}
                      className="text-sm px-3"
                    >
                      {messages.confirmDeleteButton}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="mt-4 flex justify-end gap-2">
                  <Button
                    variant="danger"
                    onClick={() => setConfirmingDelete(true)}
                    className="text-sm px-3"
                  >
                    Delete
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() => setIsEditing(true)}
                    className="text-sm px-3"
                  >
                    Edit
                  </Button>
                </div>
              )}
            </div>

            <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
              <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                {messages.goalSectionHeading}
              </h2>
              {!goalsData ? (
                <div className="flex justify-center py-2">
                  <Spinner size="sm" />
                </div>
              ) : goal?.kind === "inherited" ? (
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {messages.goalInheritedFrom(goalNameById.get(goal.goalId) ?? "")}
                </p>
              ) : goal?.kind === "direct" ? (
                <div className="flex flex-col gap-2">
                  <Link
                    to={`/goals/${goal.goalId}`}
                    className="text-sm text-blue-600 hover:text-blue-700 hover:underline dark:text-blue-400"
                  >
                    {goalNameById.get(goal.goalId) ?? ""}
                  </Link>
                  <div className="flex items-center gap-2">
                    <label htmlFor="goal-select" className="sr-only">
                      {messages.goalSectionHeading}
                    </label>
                    <select
                      id="goal-select"
                      value={String(goal.goalId)}
                      onChange={(e) => handleSetGoal(BigInt(e.target.value))}
                      className="block flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 px-3 py-2 text-sm min-h-[44px]"
                    >
                      {directGoalOptions.map((g) => (
                        <option key={String(g.id)} value={String(g.id)}>
                          {g.name}
                        </option>
                      ))}
                    </select>
                    <Button
                      variant="secondary"
                      onClick={() => handleSetGoal(undefined)}
                      className="text-sm px-3"
                    >
                      {messages.unlinkGoalButton}
                    </Button>
                  </div>
                </div>
              ) : (
                <div>
                  <p className="mb-2 text-sm text-gray-500 dark:text-gray-400">
                    {messages.noGoalLinked}
                  </p>
                  {selectableGoals.length === 0 ? (
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {messages.noGoalsToPick}
                    </p>
                  ) : (
                    <>
                      <label htmlFor="goal-select" className="sr-only">
                        {messages.goalSectionHeading}
                      </label>
                      <select
                        id="goal-select"
                        defaultValue=""
                        onChange={(e) => e.target.value && handleSetGoal(BigInt(e.target.value))}
                        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 px-3 py-2 text-sm min-h-[44px]"
                      >
                        <option value="" disabled>
                          {messages.goalPickerPlaceholder}
                        </option>
                        {selectableGoals.map((g) => (
                          <option key={String(g.id)} value={String(g.id)}>
                            {g.name}
                          </option>
                        ))}
                      </select>
                    </>
                  )}
                </div>
              )}
            </div>

            {subTasks.length > 0 && (
              <div className="rounded-xl bg-white shadow-sm dark:bg-gray-800">
                <h2 className="px-4 pt-4 pb-2 text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Sub-tasks
                </h2>
                <ul className="list-none p-0 m-0 divide-y divide-gray-100 dark:divide-gray-700">
                  {subTasks.map((sub) => {
                    const isDone = !!sub.completedAt;
                    return (
                      <li key={String(sub.id)}>
                        <button
                          className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-700/40"
                          onClick={() => navigate(`/tasks/${sub.id}`)}
                        >
                          <span
                            className={[
                              "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2",
                              isDone
                                ? "border-green-500 bg-green-500 text-white"
                                : "border-gray-400 dark:border-gray-500",
                            ].join(" ")}
                          >
                            {isDone && (
                              <svg className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                                <path
                                  fillRule="evenodd"
                                  d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
                                  clipRule="evenodd"
                                />
                              </svg>
                            )}
                          </span>
                          <span
                            className={[
                              "flex-1 text-sm leading-snug",
                              isDone
                                ? "line-through text-gray-400 dark:text-gray-500"
                                : "text-gray-900 dark:text-gray-100",
                            ].join(" ")}
                          >
                            <Markdown mode="inline">{sub.name}</Markdown>
                          </span>
                          <svg
                            className="h-4 w-4 shrink-0 text-gray-300 dark:text-gray-600"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                          >
                            <path
                              fillRule="evenodd"
                              d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                              clipRule="evenodd"
                            />
                          </svg>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}
          </div>
        )}

        {task && isEditing && (
          <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
            <h2 className="mb-4 text-base font-semibold text-gray-900 dark:text-gray-100">
              Edit task
            </h2>
            <TaskForm
              initialName={task.name}
              initialDescription={task.description ?? ""}
              submitLabel="Save"
              loading={isUpdating}
              onSubmit={handleSaveEdit}
              onCancel={() => setIsEditing(false)}
              showScheduleFields
              initialDue={timestampToIsoDay(task.due)}
              initialSnooze={timestampToIsoDay(task.snoozeUntil)}
            />
          </div>
        )}
      </main>
    </div>
  );
}

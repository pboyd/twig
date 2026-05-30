import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getTask,
  listTasks,
  updateTask,
  completeTask,
  uncompleteTask,
} from "../gen/task/v1/task-TaskService_connectquery";
import { ConnectError, Code } from "@connectrpc/connect";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { Button } from "../components/Button";
import { TaskForm } from "../components/TaskForm";
import { messages } from "../theme/messages";
import { buildUpdatePayload } from "../lib/updatePayload";

export default function TaskDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const taskId = id ? BigInt(id) : undefined;

  const [isEditing, setIsEditing] = useState(false);
  const [toggleError, setToggleError] = useState<string | null>(null);

  const { data, isLoading, isError, error } = useQuery(
    getTask,
    taskId !== undefined ? { id: taskId } : undefined,
    { enabled: taskId !== undefined }
  );

  const { data: allTasksData } = useQuery(listTasks, {});
  const subTasks = (allTasksData?.tasks ?? []).filter(
    (t) => t.parentId !== undefined && t.parentId === taskId
  );

  const { mutateAsync: doUpdate, isPending: isUpdating } = useMutation(updateTask);
  const { mutateAsync: doComplete, isPending: isCompleting } = useMutation(completeTask);
  const { mutateAsync: doUncomplete, isPending: isUncompleting } = useMutation(uncompleteTask);

  const task = data?.task;
  const is404 =
    isError &&
    error instanceof ConnectError &&
    error.code === Code.NotFound;

  async function handleSaveEdit(editedName: string, editedDesc: string) {
    if (!task) return;
    const payload = buildUpdatePayload(task, editedName, editedDesc);
    await doUpdate(payload);
    await queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: getTask, input: { id: taskId }, cardinality: "finite" }),
    });
    await queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
    });
    setIsEditing(false);
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

  const isToggling = isCompleting || isUncompleting;

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
                    {task.name}
                  </h1>
                  {task.description && (
                    <p className="mt-2 text-sm text-gray-600 dark:text-gray-400 whitespace-pre-wrap">
                      {task.description}
                    </p>
                  )}
                  <p className="mt-3 text-xs text-gray-400 dark:text-gray-500">
                    {task.completedAt ? "Completed" : "Incomplete"}
                  </p>
                </div>
              </div>
              <div className="mt-4 flex justify-end">
                <Button
                  variant="secondary"
                  onClick={() => setIsEditing(true)}
                  className="text-sm px-3"
                >
                  Edit
                </Button>
              </div>
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
                            {sub.name}
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
            />
          </div>
        )}
      </main>
    </div>
  );
}

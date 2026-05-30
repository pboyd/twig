import { useState } from "react";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listTasks,
  createTask,
} from "../gen/task/v1/task-TaskService_connectquery";
import { buildTree } from "../lib/tree";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { TreeRow } from "../components/TreeRow";
import { TaskForm } from "../components/TaskForm";
import { Button } from "../components/Button";
import { messages } from "../theme/messages";

export default function TaskTreePage() {
  const queryClient = useQueryClient();
  const [showAddForm, setShowAddForm] = useState(false);

  const { data, isLoading, isError, error, refetch } = useQuery(listTasks, {});
  const { mutateAsync: doCreateTask, isPending } = useMutation(createTask);

  async function handleAddTask(name: string, description: string) {
    await doCreateTask({ name, description });
    await queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
    });
    setShowAddForm(false);
  }

  const tasks = data?.tasks ?? [];
  const tree = buildTree(tasks);

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-0 pb-8">
        <div className="flex items-center justify-between px-4 py-3">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Tasks
          </h1>
          <Button onClick={() => setShowAddForm((v) => !v)} className="text-sm px-3">
            + Add task
          </Button>
        </div>

        {showAddForm && (
          <div className="mx-4 mb-4 rounded-md border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <TaskForm
              onSubmit={handleAddTask}
              onCancel={() => setShowAddForm(false)}
              loading={isPending}
            />
          </div>
        )}

        {isLoading && (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        )}

        {isError && (
          <div className="mx-4">
            <ErrorBanner
              message={error?.message ?? messages.connectivityError}
              onRetry={() => refetch()}
            />
          </div>
        )}

        {!isLoading && !isError && tasks.length === 0 && (
          <div className="px-4 py-12 text-center text-gray-500 dark:text-gray-400">
            <p>{messages.emptyTaskList}</p>
          </div>
        )}

        {!isLoading && !isError && tree.length > 0 && (
          <ul className="list-none p-0 m-0 border-t border-gray-100 dark:border-gray-800/60">
            {tree.map((node) => (
              <TreeRow key={String(node.task.id)} node={node} />
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}

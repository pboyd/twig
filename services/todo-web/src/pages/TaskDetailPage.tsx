import { useParams, useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { getTask } from "../gen/task/v1/task-TaskService_connectquery";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { Button } from "../components/Button";
import { messages } from "../theme/messages";
import { ConnectError, Code } from "@connectrpc/connect";

export default function TaskDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const taskId = id ? BigInt(id) : undefined;

  const { data, isLoading, isError, error } = useQuery(
    getTask,
    taskId !== undefined ? { id: taskId } : undefined,
    { enabled: taskId !== undefined }
  );

  const task = data?.task;
  const is404 =
    isError &&
    error instanceof ConnectError &&
    error.code === Code.NotFound;

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-4 py-4">
        <Button
          variant="secondary"
          onClick={() => navigate(-1)}
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

        {task && (
          <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
            <h1 className="mb-2 text-xl font-semibold text-gray-900 dark:text-gray-100">
              {task.name}
            </h1>
            {task.description && (
              <p className="text-sm text-gray-600 dark:text-gray-400 whitespace-pre-wrap">
                {task.description}
              </p>
            )}
            <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
              {task.completedAt ? "Completed" : "Incomplete"}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { ConnectError, Code } from "@connectrpc/connect";
import {
  getGoal,
  listGoals,
  listGoalStatusUpdates,
  addGoalStatusUpdate,
  updateGoalStatusUpdate,
  deleteGoalStatusUpdate,
} from "../gen/goal/v1/goal-GoalService_connectquery";
import { GoalState } from "../gen/goal/v1/goal_pb";
import type { StatusUpdate } from "../gen/goal/v1/goal_pb";
import { listTasks } from "../gen/task/v1/task-TaskService_connectquery";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { Button } from "../components/Button";
import { Markdown } from "../components/Markdown";
import { StatusUpdateList } from "../components/StatusUpdateList";
import { StatusUpdateForm } from "../components/StatusUpdateForm";
import { formatTimestamp, formatDueDate } from "../lib/formatTimestamp";
import { useToast } from "../context/ToastProvider";
import { messages } from "../theme/messages";

const STATE_LABELS: Partial<Record<GoalState, string>> = {
  [GoalState.COMMITTED]: "Committed",
  [GoalState.INCUBATING]: "Incubating",
  [GoalState.COMPLETED]: "Completed",
  [GoalState.ARCHIVED]: "Archived",
};

export default function GoalDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const toast = useToast();
  const goalId = id ? BigInt(id) : undefined;

  const [showAddForm, setShowAddForm] = useState(false);
  const [editingUpdate, setEditingUpdate] = useState<StatusUpdate | null>(null);
  const [bannerError, setBannerError] = useState<string | null>(null);

  const { data, isLoading, isError, error } = useQuery(
    getGoal,
    goalId !== undefined ? { id: goalId } : undefined,
    { enabled: goalId !== undefined },
  );

  const { data: updatesData, isLoading: isUpdatesLoading, isError: isUpdatesError } = useQuery(
    listGoalStatusUpdates,
    goalId !== undefined ? { goalId } : undefined,
    { enabled: goalId !== undefined },
  );

  const { data: allTasksData } = useQuery(listTasks, {});

  const { mutateAsync: doAdd, isPending: isAdding } = useMutation(addGoalStatusUpdate);
  const { mutateAsync: doUpdate, isPending: isUpdating } = useMutation(updateGoalStatusUpdate);
  const { mutateAsync: doDelete } = useMutation(deleteGoalStatusUpdate);

  const goal = data?.goal;
  const updates = updatesData?.updates ?? [];
  const is404 =
    isError &&
    error instanceof ConnectError &&
    error.code === Code.NotFound;

  const associatedTasks = (allTasksData?.tasks ?? []).filter(
    (t) => t.goalId !== undefined && t.goalId === goalId,
  );

  function invalidateAll() {
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listGoalStatusUpdates, input: { goalId }, cardinality: "finite" }),
    });
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: getGoal, input: { id: goalId }, cardinality: "finite" }),
    });
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listGoals, input: {}, cardinality: "finite" }),
    });
  }

  async function handleAdd(body: string) {
    if (!goalId) return;
    setBannerError(null);
    try {
      await doAdd({ goalId, body });
      invalidateAll();
      toast.show(messages.statusSaved);
      setShowAddForm(false);
    } catch {
      setBannerError(messages.statusSaveError);
    }
  }

  async function handleEdit(update: StatusUpdate) {
    setEditingUpdate(update);
  }

  async function handleEditSave(body: string) {
    if (!editingUpdate) return;
    setBannerError(null);
    try {
      await doUpdate({ id: editingUpdate.id, body });
      invalidateAll();
      toast.show(messages.statusEdited);
      setEditingUpdate(null);
    } catch {
      setBannerError(messages.statusSaveError);
    }
  }

  async function handleDelete(id: bigint) {
    if (!confirm(messages.deleteUpdateConfirm)) return;
    setBannerError(null);
    try {
      await doDelete({ id });
      invalidateAll();
      toast.show(messages.statusDeleted);
    } catch {
      setBannerError(messages.statusDeleteError);
    }
  }

  const stateLabel = goal
    ? STATE_LABELS[goal.state] ?? String(goal.state)
    : "";

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-4 py-4">
        <Button
          variant="secondary"
          onClick={() => navigate("/goals")}
          className="mb-4 text-sm px-3"
        >
          ← Back
        </Button>

        {bannerError && (
          <div className="mb-4">
            <ErrorBanner message={bannerError} />
          </div>
        )}

        {isLoading && (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        )}

        {is404 && (
          <div className="flex flex-col items-center justify-center py-16 px-4 text-center gap-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {messages.goalNotFound}
            </p>
            <Button
              variant="secondary"
              onClick={() => navigate("/goals")}
              className="text-sm px-3"
            >
              ← Back to goals
            </Button>
          </div>
        )}

        {isError && !is404 && (
          <ErrorBanner message={messages.goalDetailLoadError} />
        )}

        {goal && (
          <div className="space-y-4">
            <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                    <Markdown mode="inline">{goal.name}</Markdown>
                  </h1>
                  {goal.description && (
                    <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                      <Markdown mode="block">{goal.description}</Markdown>
                    </div>
                  )}
                  <div className="mt-3 flex items-center gap-3">
                    {goal.due && (
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        Due {formatDueDate(goal.due)}
                      </span>
                    )}
                    <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                      {stateLabel}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Latest status */}
            {updates.length > 0 && !editingUpdate && !showAddForm && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Latest status
                </h2>
                <div className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                  {formatTimestamp(updates[0].createdAt)}
                </div>
                <div className="prose-sm text-sm text-gray-700 dark:text-gray-300">
                  <Markdown mode="block">{updates[0].body}</Markdown>
                </div>
              </div>
            )}

            {/* Status updates load error */}
            {isUpdatesError && !showAddForm && !editingUpdate && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <ErrorBanner message={messages.statusLoadError} />
              </div>
            )}

            {/* No updates message */}
            {updates.length === 0 && !isUpdatesError && !showAddForm && !editingUpdate && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {messages.noStatusUpdates}
                </p>
              </div>
            )}

            {/* Add status update form */}
            {showAddForm && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Add status update
                </h2>
                <StatusUpdateForm
                  submitLabel="Save"
                  loading={isAdding}
                  onSubmit={handleAdd}
                  onCancel={() => setShowAddForm(false)}
                />
              </div>
            )}

            {/* Edit status update form */}
            {editingUpdate && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Edit status update
                </h2>
                <StatusUpdateForm
                  initialBody={editingUpdate.body}
                  submitLabel="Update"
                  loading={isUpdating}
                  onSubmit={handleEditSave}
                  onCancel={() => setEditingUpdate(null)}
                />
              </div>
            )}

            {/* Add button */}
            {!showAddForm && !editingUpdate && (
              <Button
                variant="secondary"
                onClick={() => setShowAddForm(true)}
                className="text-sm px-3"
              >
                Add status update
              </Button>
            )}

            {/* Status history */}
            {updates.length > 0 && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  History
                </h2>
                {isUpdatesLoading ? (
                  <div className="flex justify-center py-4">
                    <Spinner size="sm" />
                  </div>
                ) : (
                  <StatusUpdateList
                    updates={updates}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                  />
                )}
              </div>
            )}

            {/* Associated tasks */}
            {associatedTasks.length > 0 && (
              <div className="rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
                <h2 className="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Associated tasks
                </h2>
                <div className="flex flex-col gap-1">
                  {associatedTasks.map((task) => (
                    <button
                      key={task.id}
                      type="button"
                      onClick={() => navigate(`/tasks/${task.id}`)}
                      className="text-left text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
                    >
                      <Markdown mode="inline">{task.name}</Markdown>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

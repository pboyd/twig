import { useState, useRef, useEffect, useCallback } from "react";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listTasks,
  createTask,
  reorderTask,
} from "../gen/task/v1/task-TaskService_connectquery";
import { DndContext, closestCenter, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { buildTree, findSiblingIds } from "../lib/tree";
import type { TaskNode } from "../lib/tree";
import { filterTree } from "../lib/filterTree";
import { reorderAnchor, type ReorderAnchor } from "../lib/reorderAnchor";
import { readShowCompleted, writeShowCompleted } from "../lib/showCompletedPref";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { TreeRow } from "../components/TreeRow";
import { EmptyState } from "../components/EmptyState";
import { TaskForm } from "../components/TaskForm";
import { Button } from "../components/Button";
import { messages } from "../theme/messages";
import { useToast } from "../context/ToastProvider";
import { useAddTaskToPlan } from "../hooks/useAddTaskToPlan";

const EXPANDED_STORAGE_KEY = "twig-expanded-tasks";

function readExpandedIds(): Set<bigint> | null {
  try {
    const stored = sessionStorage.getItem(EXPANDED_STORAGE_KEY);
    if (stored !== null) {
      const arr = JSON.parse(stored) as string[];
      return new Set(arr.map(BigInt));
    }
  } catch {}
  return null;
}

function writeExpandedIds(ids: Set<bigint>) {
  try {
    sessionStorage.setItem(EXPANDED_STORAGE_KEY, JSON.stringify([...ids].map(String)));
  } catch {}
}

function collectParentIds(nodes: TaskNode[]): Set<bigint> {
  const ids = new Set<bigint>();
  function walk(list: TaskNode[]) {
    for (const n of list) {
      if (n.children.length > 0) {
        ids.add(n.task.id);
        walk(n.children);
      }
    }
  }
  walk(nodes);
  return ids;
}

export default function TaskTreePage() {
  const queryClient = useQueryClient();
  const [showAddForm, setShowAddForm] = useState(false);
  const [expandedIds, setExpandedIds] = useState<Set<bigint>>(new Set());
  const [reorderError, setReorderError] = useState<string | null>(null);
  const initializedRef = useRef(false);
  const { show: showToast } = useToast();

  const { data, isLoading, isError, error, refetch } = useQuery(listTasks, {});
  const { mutateAsync: doCreateTask, isPending } = useMutation(createTask);
  const { mutateAsync: doReorderTask } = useMutation(reorderTask);
  const { addToPlan } = useAddTaskToPlan(messages.addedToPlanPartialFail);

  const [showCompleted, setShowCompleted] = useState(readShowCompleted);

  const tasks = data?.tasks ?? [];
  const tree = buildTree(tasks);
  const filteredTree = filterTree(tree, showCompleted);

  const listTasksKey = createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" });

  // Initialize expand state once when tree data arrives
  useEffect(() => {
    if (!initializedRef.current && tree.length > 0) {
      initializedRef.current = true;
      const stored = readExpandedIds();
      if (stored !== null) {
        setExpandedIds(stored);
      } else {
        const defaultExpanded = collectParentIds(tree);
        setExpandedIds(defaultExpanded);
        writeExpandedIds(defaultExpanded);
      }
    }
  }, [tree]);

  function toggleExpand(id: bigint) {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      writeExpandedIds(next);
      return next;
    });
  }

  async function handleAddTask(name: string, description: string, planDay?: string) {
    let createResp;
    try {
      createResp = await doCreateTask({ name, description });
    } catch {
      showToast(messages.addFailed, "error");
      return;
    }
    const newTaskId = createResp.task?.id;

    if (planDay && newTaskId) {
      await addToPlan(planDay, newTaskId);
    }

    await queryClient.invalidateQueries({ queryKey: listTasksKey });
    setShowAddForm(false);
  }

  const handleReorder = useCallback(async (taskId: bigint, anchor: ReorderAnchor) => {
    setReorderError(null);
    try {
      await doReorderTask({ taskId, anchor });
      await queryClient.invalidateQueries({ queryKey: listTasksKey });
    } catch {
      setReorderError(messages.reorderError);
    }
  }, [doReorderTask, queryClient, listTasksKey]);

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeId = BigInt(active.id as string);
    const overId = BigInt(over.id as string);

    const siblingIds = findSiblingIds(filteredTree, activeId);
    if (!siblingIds) return;

    // Ensure over item is in the same sibling group (reorder-only, no re-parenting)
    if (!siblingIds.includes(overId)) return;

    const anchor = reorderAnchor(activeId, overId, siblingIds);
    if (!anchor) return;

    void handleReorder(activeId, anchor);
  }

  const rootSiblingIds = filteredTree.map((n) => String(n.task.id));

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-0 pb-8">
        <div className="flex items-center justify-between px-4 py-3">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Tasks
          </h1>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              onClick={() => setShowCompleted((v) => { const next = !v; writeShowCompleted(next); return next; })}
              className="text-sm px-3"
            >
              {showCompleted ? "Show only pending" : "Show all"}
            </Button>
            <Button onClick={() => setShowAddForm((v) => !v)} className="text-sm px-3">
              + Add task
            </Button>
          </div>
        </div>

        {showAddForm && (
          <div className="mx-4 mb-4 rounded-md border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <TaskForm
              onSubmit={handleAddTask}
              onCancel={() => setShowAddForm(false)}
              loading={isPending}
              showPlanControl
            />
          </div>
        )}

        {reorderError && (
          <div className="mx-4 mb-2">
            <ErrorBanner message={reorderError} onRetry={() => setReorderError(null)} />
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
          <EmptyState onAddTask={() => setShowAddForm(true)} />
        )}

        {!isLoading && !isError && tasks.length > 0 && filteredTree.length === 0 && (
          <div className="mx-4 mt-8 text-center text-gray-500 dark:text-gray-400">
            <p>{messages.allHidden}</p>
            <Button
              variant="secondary"
              onClick={() => { writeShowCompleted(true); setShowCompleted(true); }}
              className="mt-3 text-sm px-3"
            >
              Show all
            </Button>
          </div>
        )}

        {!isLoading && !isError && filteredTree.length > 0 && (
          <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={rootSiblingIds} strategy={verticalListSortingStrategy}>
              <ul className="list-none p-0 m-0 border-t border-gray-100 dark:border-gray-800/60">
                {filteredTree.map((node) => (
                  <TreeRow
                    key={String(node.task.id)}
                    node={node}
                    expandedIds={expandedIds}
                    onToggleExpand={toggleExpand}
                    onReorder={handleReorder}
                  />
                ))}
              </ul>
            </SortableContext>
          </DndContext>
        )}
      </main>
    </div>
  );
}

import { useState, useRef, useEffect } from "react";
import { useQuery, useMutation, createConnectQueryKey } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listTasks,
  createTask,
} from "../gen/task/v1/task-TaskService_connectquery";
import { buildTree } from "../lib/tree";
import type { TaskNode } from "../lib/tree";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { TreeRow } from "../components/TreeRow";
import { EmptyState } from "../components/EmptyState";
import { TaskForm } from "../components/TaskForm";
import { Button } from "../components/Button";
import { messages } from "../theme/messages";

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
  const initializedRef = useRef(false);

  const { data, isLoading, isError, error, refetch } = useQuery(listTasks, {});
  const { mutateAsync: doCreateTask, isPending } = useMutation(createTask);

  const tasks = data?.tasks ?? [];
  const tree = buildTree(tasks);

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

  async function handleAddTask(name: string, description: string) {
    await doCreateTask({ name, description });
    await queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
    });
    setShowAddForm(false);
  }

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
          <EmptyState onAddTask={() => setShowAddForm(true)} />
        )}

        {!isLoading && !isError && tree.length > 0 && (
          <ul className="list-none p-0 m-0 border-t border-gray-100 dark:border-gray-800/60">
            {tree.map((node) => (
              <TreeRow
                key={String(node.task.id)}
                node={node}
                expandedIds={expandedIds}
                onToggleExpand={toggleExpand}
              />
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}

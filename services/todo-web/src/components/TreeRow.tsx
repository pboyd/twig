import { useState } from "react";
import { useNavigate } from "react-router";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  createTask,
  listTasks,
} from "../gen/task/v1/task-TaskService_connectquery";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import type { TaskNode } from "../lib/tree";
import { TaskForm } from "./TaskForm";

interface TreeRowProps {
  node: TaskNode;
}

export function TreeRow({ node }: TreeRowProps) {
  const { task, children, depth } = node;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showSubForm, setShowSubForm] = useState(false);

  const { mutateAsync: doCreateTask, isPending } = useMutation(createTask);

  async function handleAddSubTask(name: string, description: string) {
    await doCreateTask({ name, description, parentId: task.id });
    await queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" }),
    });
    setShowSubForm(false);
  }

  const indentRem = depth * 1;
  const isComplete = !!task.completedAt;

  return (
    <li>
      <div
        className="flex items-center gap-1 px-4 border-b border-gray-100 dark:border-gray-800/60"
        style={{ paddingLeft: `${1 + indentRem}rem` }}
      >
        {/* Completion indicator — 44px touch target, 20px visual circle */}
        <button
          aria-label={isComplete ? "Completed" : "Incomplete"}
          className="flex h-11 w-11 shrink-0 items-center justify-center -ml-2 rounded"
          onClick={() => navigate(`/tasks/${task.id}`)}
        >
          <span
            className={[
              "flex h-5 w-5 items-center justify-center rounded-full border-2",
              isComplete
                ? "border-green-500 bg-green-500 text-white"
                : "border-gray-400 dark:border-gray-500",
            ].join(" ")}
          >
            {isComplete && (
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

        {/* Task name — tappable, wraps rather than truncates */}
        <button
          className="flex-1 py-3 text-left text-sm leading-snug text-gray-900 dark:text-gray-100"
          onClick={() => navigate(`/tasks/${task.id}`)}
        >
          <span className={isComplete ? "line-through text-gray-400 dark:text-gray-500" : ""}>
            {task.name}
          </span>
        </button>

        {/* Add sub-task — 44px touch target, small icon */}
        <button
          aria-label="Add sub-task"
          title="Add sub-task"
          className="flex h-11 w-11 shrink-0 items-center justify-center -mr-2 rounded text-gray-400 hover:text-blue-500 dark:hover:text-blue-400"
          onClick={() => setShowSubForm((v) => !v)}
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
          </svg>
        </button>
      </div>

      {showSubForm && (
        <div
          className="px-4 pb-3 pt-2 bg-gray-50 dark:bg-gray-800/40 border-b border-gray-100 dark:border-gray-800/60"
          style={{ paddingLeft: `${1 + indentRem + 1}rem` }}
        >
          <TaskForm
            onSubmit={handleAddSubTask}
            onCancel={() => setShowSubForm(false)}
            loading={isPending}
            submitLabel="Add sub-task"
          />
        </div>
      )}

      {children.length > 0 && (
        <ul className="list-none p-0 m-0">
          {children.map((child) => (
            <TreeRow key={String(child.task.id)} node={child} />
          ))}
        </ul>
      )}
    </li>
  );
}

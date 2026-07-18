import { useState } from "react";
import { useNavigate } from "react-router";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  createTask,
  listTasks,
  completeTask,
  uncompleteTask,
} from "../gen/task/v1/task-TaskService_connectquery";
import {
  addPlanTask,
  listPlanEntries,
} from "../gen/plan/v1/plan-PlanService_connectquery";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { ConnectError, Code } from "@connectrpc/connect";
import { useSortable, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { TaskNode } from "../lib/tree";
import type { ReorderAnchor } from "../lib/reorderAnchor";
import { TaskForm } from "./TaskForm";
import { AddToPlanControl } from "./AddToPlanControl";
import { CompletionToggle } from "./CompletionToggle";
import { messages } from "../theme/messages";
import { Markdown } from "./Markdown";
import { useToast } from "../context/ToastProvider";
import { dayPickerLabel } from "../lib/planDays";

interface TreeRowProps {
  node: TaskNode;
  expandedIds: Set<bigint>;
  onToggleExpand: (id: bigint) => void;
  onReorder: (taskId: bigint, anchor: ReorderAnchor) => void;
}

export function TreeRow({ node, expandedIds, onToggleExpand, onReorder }: TreeRowProps) {
  const { task, children, depth } = node;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showSubForm, setShowSubForm] = useState(false);
  const [toggleError, setToggleError] = useState<string | null>(null);
  const { show: showToast } = useToast();

  const { mutateAsync: doCreateTask, isPending: isCreating } = useMutation(createTask);
  const { mutateAsync: doAddPlanTask } = useMutation(addPlanTask);
  const { mutateAsync: doComplete, isPending: isCompleting } = useMutation(completeTask);
  const { mutateAsync: doUncomplete, isPending: isUncompleting } = useMutation(uncompleteTask);

  const listTasksKey = createConnectQueryKey({ schema: listTasks, input: {}, cardinality: "finite" });

  function planEntriesKey(day: string) {
    return createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" });
  }

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: String(task.id) });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : undefined,
  };

  async function handleAddSubTask(name: string, description: string, planDay?: string) {
    const createResp = await doCreateTask({ name, description, parentId: task.id });
    const newTaskId = createResp.task?.id;

    if (planDay && newTaskId) {
      try {
        await doAddPlanTask({ day: planDay, taskId: newTaskId, durationMinute: 0 });
        const label = dayPickerLabel(planDay);
        showToast(label === "Today" ? messages.addedToToday : messages.addedToDay(label), "success");
        await queryClient.invalidateQueries({ queryKey: planEntriesKey(planDay) });
      } catch (err) {
        if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
          showToast(messages.alreadyOnPlan, "error");
        } else {
          showToast(messages.addedToPlanPartialFail, "error");
        }
      }
    }

    await queryClient.invalidateQueries({ queryKey: listTasksKey });
    setShowSubForm(false);
  }

  async function handleToggleComplete() {
    setToggleError(null);
    try {
      if (task.completedAt) {
        await doUncomplete({ id: task.id });
      } else {
        await doComplete({ id: task.id });
      }
      await queryClient.invalidateQueries({ queryKey: listTasksKey });
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

  const indentRem = depth * 1;
  const isComplete = !!task.completedAt;
  const isSnoozed = (() => {
    if (!task.snoozeUntil) return false;
    const snooze = new Date(Number(task.snoozeUntil.seconds) * 1000);
    const today = new Date();
    if (snooze.getUTCFullYear() !== today.getFullYear()) return snooze.getUTCFullYear() > today.getFullYear();
    if (snooze.getUTCMonth() !== today.getMonth()) return snooze.getUTCMonth() > today.getMonth();
    return snooze.getUTCDate() > today.getDate();
  })();
  const hasChildren = children.length > 0;
  const isExpanded = expandedIds.has(task.id);
  const isToggling = isCompleting || isUncompleting;

  const childSiblingIds = children.map((n) => String(n.task.id));

  return (
    <li ref={setNodeRef} style={style}>
      <div
        className="flex items-center gap-1 px-4 border-b border-gray-100 dark:border-gray-800/60"
        style={{ paddingLeft: `${1 + indentRem}rem` }}
      >
        {/* Drag handle */}
        <button
          aria-label={messages.dragHandleLabel}
          className="flex h-11 w-6 shrink-0 items-center justify-center -ml-2 rounded cursor-grab active:cursor-grabbing text-gray-300 hover:text-gray-500 dark:text-gray-600 dark:hover:text-gray-400 touch-none"
          {...attributes}
          {...listeners}
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M7 4a1 1 0 1 1-2 0 1 1 0 0 1 2 0zm6 0a1 1 0 1 1-2 0 1 1 0 0 1 2 0zM7 10a1 1 0 1 1-2 0 1 1 0 0 1 2 0zm6 0a1 1 0 1 1-2 0 1 1 0 0 1 2 0zM7 16a1 1 0 1 1-2 0 1 1 0 0 1 2 0zm6 0a1 1 0 1 1-2 0 1 1 0 0 1 2 0z" />
          </svg>
        </button>

        {/* Completion toggle — 44px touch target */}
        <CompletionToggle
          completed={isComplete}
          disabled={isToggling}
          onToggle={handleToggleComplete}
        />

        {/* Task name — tappable, wraps rather than truncates */}
        <button
          className="flex-1 py-3 text-left text-sm leading-snug text-gray-900 dark:text-gray-100"
          onClick={() => navigate(`/tasks/${task.id}`)}
        >
          <span className={isComplete ? "line-through text-gray-400 dark:text-gray-500" : ""}>
            <Markdown mode="inline">{task.name}</Markdown>
          </span>
          {isSnoozed && <span className="ml-1 text-base" aria-label="snoozed">💤</span>}
        </button>

        {/* Expand/collapse chevron — only for nodes with children */}
        {hasChildren && (
          <button
            aria-label={isExpanded ? "Collapse" : "Expand"}
            className="flex h-11 w-8 shrink-0 items-center justify-center rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            onClick={() => onToggleExpand(task.id)}
          >
            <svg
              className={[
                "h-4 w-4 transition-transform duration-150",
                isExpanded ? "rotate-90" : "rotate-0",
              ].join(" ")}
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
        )}

        {/* Add sub-task — 44px touch target */}
        <button
          aria-label="Add sub-task"
          title="Add sub-task"
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded text-gray-400 hover:text-blue-500 dark:hover:text-blue-400"
          onClick={() => setShowSubForm((v) => !v)}
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
          </svg>
        </button>

        {/* Add to plan */}
        <div className="-mr-2">
          <AddToPlanControl taskId={task.id} />
        </div>
      </div>

      {toggleError && (
        <div
          className="px-4 py-2 text-sm text-amber-700 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400 border-b border-gray-100 dark:border-gray-800/60"
          style={{ paddingLeft: `${1 + indentRem}rem` }}
        >
          {toggleError}
        </div>
      )}

      {showSubForm && (
        <div
          className="px-4 pb-3 pt-2 bg-gray-50 dark:bg-gray-800/40 border-b border-gray-100 dark:border-gray-800/60"
          style={{ paddingLeft: `${1 + indentRem + 1}rem` }}
        >
          <TaskForm
            onSubmit={handleAddSubTask}
            onCancel={() => setShowSubForm(false)}
            loading={isCreating}
            submitLabel="Add sub-task"
            showPlanControl
          />
        </div>
      )}

      {hasChildren && isExpanded && (
        <SortableContext items={childSiblingIds} strategy={verticalListSortingStrategy}>
          <ul className="list-none p-0 m-0">
            {children.map((child) => (
              <TreeRow
                key={String(child.task.id)}
                node={child}
                expandedIds={expandedIds}
                onToggleExpand={onToggleExpand}
                onReorder={onReorder}
              />
            ))}
          </ul>
        </SortableContext>
      )}
    </li>
  );
}

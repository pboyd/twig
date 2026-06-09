import { useState } from "react";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { ConnectError, Code } from "@connectrpc/connect";
import {
  addPlanTask,
  listPlanEntries,
} from "../gen/plan/v1/plan-PlanService_connectquery";
import { useToast } from "../context/ToastProvider";
import { messages } from "../theme/messages";
import { todayIso, tomorrowIso, dayPickerLabel } from "../lib/planDays";
import { Button } from "./Button";

interface AddToPlanControlProps {
  taskId: bigint;
}

export function AddToPlanControl({ taskId }: AddToPlanControlProps) {
  const [showPicker, setShowPicker] = useState(false);
  const [pickedDate, setPickedDate] = useState("");
  const { show } = useToast();
  const queryClient = useQueryClient();

  const { mutateAsync: doAdd, isPending } = useMutation(addPlanTask);

  async function addToDay(day: string) {
    try {
      await doAdd({ day, taskId, durationMinute: 0 });
      await queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" }),
      });
      const label = dayPickerLabel(day);
      show(label === "Today" ? messages.addedToToday : messages.addedToDay(label), "success");
      setShowPicker(false);
      setPickedDate("");
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
        show(messages.alreadyOnPlan, "error");
      } else {
        show(messages.connectivityError, "error");
      }
    }
  }

  return (
    <div className="relative flex items-center">
      {/* Add to today — single tap */}
      <button
        aria-label="Add to today's plan"
        title="Add to today's plan"
        disabled={isPending}
        className="flex h-11 w-11 shrink-0 items-center justify-center rounded text-gray-400 hover:text-blue-500 dark:hover:text-blue-400 disabled:opacity-50"
        onClick={(e) => {
          e.stopPropagation();
          addToDay(todayIso());
        }}
      >
        <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path
            fillRule="evenodd"
            d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zM4 8h12v8H4V8zm4 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {/* Other day picker — toggle */}
      <button
        aria-label="Add to another day"
        title="Add to another day"
        disabled={isPending}
        className="flex h-11 w-7 shrink-0 items-center justify-center rounded text-gray-300 hover:text-blue-400 dark:hover:text-blue-400 disabled:opacity-50 -ml-1"
        onClick={(e) => {
          e.stopPropagation();
          setShowPicker((v) => !v);
        }}
      >
        <svg className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path
            fillRule="evenodd"
            d="M5.22 8.22a.75.75 0 011.06 0L10 11.94l3.72-3.72a.75.75 0 111.06 1.06l-4.25 4.25a.75.75 0 01-1.06 0L5.22 9.28a.75.75 0 010-1.06z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {/* Day picker popover */}
      {showPicker && (
        <div
          className="absolute right-0 top-full mt-1 z-10 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg p-3 flex flex-col gap-2 min-w-[200px]"
        >
          <Button
            variant="secondary"
            className="text-sm w-full"
            onClick={(e) => {
              e.stopPropagation();
              addToDay(tomorrowIso());
            }}
          >
            Tomorrow
          </Button>
          <input
            type="date"
            aria-label="Pick a date"
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 px-3 py-2 text-sm min-h-[44px]"
            value={pickedDate}
            onChange={(e) => setPickedDate(e.target.value)}
          />
          <Button
            variant="primary"
            className="text-sm w-full"
            disabled={!pickedDate}
            onClick={(e) => {
              e.stopPropagation();
              if (pickedDate) addToDay(pickedDate);
            }}
          >
            Add
          </Button>
        </div>
      )}
    </div>
  );
}

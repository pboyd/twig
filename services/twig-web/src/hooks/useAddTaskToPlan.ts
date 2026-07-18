import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { ConnectError, Code } from "@connectrpc/connect";
import { addPlanTask, listPlanEntries } from "../gen/plan/v1/plan-PlanService_connectquery";
import { useToast } from "../context/ToastProvider";
import { messages } from "../theme/messages";
import { dayPickerLabel } from "../lib/planDays";

/** Query key for a single day's plan entries — shared so callers invalidate consistently. */
export function planEntriesKey(day: string) {
  return createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" });
}

/**
 * Adds a task to a day's plan and handles the toast/invalidation/error
 * bookkeeping that every "add to plan" entry point needs.
 *
 * `genericErrorMessage` lets each caller phrase the non-FailedPrecondition
 * failure case in the way that fits its flow (e.g. a partial-failure notice
 * right after create, vs. a plain connectivity error from the plan picker).
 */
export function useAddTaskToPlan(genericErrorMessage: string) {
  const { show: showToast } = useToast();
  const queryClient = useQueryClient();
  const { mutateAsync: doAddPlanTask, isPending } = useMutation(addPlanTask);

  async function addToPlan(day: string, taskId: bigint): Promise<boolean> {
    try {
      await doAddPlanTask({ day, taskId, durationMinute: 0 });
      const label = dayPickerLabel(day);
      showToast(label === "Today" ? messages.addedToToday : messages.addedToDay(label), "success");
      await queryClient.invalidateQueries({ queryKey: planEntriesKey(day) });
      return true;
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
        showToast(messages.alreadyOnPlan, "error");
      } else {
        showToast(genericErrorMessage, "error");
      }
      return false;
    }
  }

  return { addToPlan, isPending };
}

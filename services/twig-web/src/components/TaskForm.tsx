import { useState, type FormEvent } from "react";
import { Field } from "./Field";
import { Button } from "./Button";
import { messages } from "../theme/messages";
import { todayIso, tomorrowIso } from "../lib/planDays";

type PlanChoice = "none" | "today" | "tomorrow" | "date";

interface TaskFormProps {
  onSubmit: (name: string, description: string, planDay?: string) => Promise<void>;
  onCancel?: () => void;
  loading?: boolean;
  initialName?: string;
  initialDescription?: string;
  submitLabel?: string;
  showPlanControl?: boolean;
}

export function TaskForm({
  onSubmit,
  onCancel,
  loading = false,
  initialName = "",
  initialDescription = "",
  submitLabel = "Add task",
  showPlanControl = false,
}: TaskFormProps) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [nameError, setNameError] = useState<string | null>(null);
  const [planChoice, setPlanChoice] = useState<PlanChoice>("none");
  const [planDate, setPlanDate] = useState("");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.emptyNameValidation);
      return;
    }
    setNameError(null);

    let planDay: string | undefined;
    switch (planChoice) {
      case "today":
        planDay = todayIso();
        break;
      case "tomorrow":
        planDay = tomorrowIso();
        break;
      case "date":
        if (planDate) planDay = planDate;
        break;
      case "none":
      default:
        break;
    }

    await onSubmit(trimmed, description.trim(), planDay);
  }

  const planChoices: { value: PlanChoice; label: string }[] = [
    { value: "none", label: "No plan" },
    { value: "today", label: "Today" },
    { value: "tomorrow", label: "Tomorrow" },
  ];

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <Field
        id="task-name"
        label="Title"
        type="text"
        value={name}
        onChange={(e) => {
          setName(e.target.value);
          if (nameError) setNameError(null);
        }}
        error={nameError ?? undefined}
        placeholder="What needs doing?"
        required
      />
      <Field
        id="task-description"
        label="Description"
        multiline
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="Optional details…"
      />

      {showPlanControl && (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Add to plan
          </label>
          <div
            role="radiogroup"
            aria-label="Add to plan"
            className="flex gap-0 rounded-md border border-gray-200 dark:border-gray-700 overflow-hidden"
          >
            {planChoices.map((choice) => (
              <button
                key={choice.value}
                type="button"
                role="radio"
                aria-checked={planChoice === choice.value}
                className={[
                  "flex-1 px-3 py-2 text-sm min-h-[44px] border-r border-gray-200 dark:border-gray-700 last:border-r-0 transition-colors",
                  planChoice === choice.value
                    ? "bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300 font-medium"
                    : "bg-white text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700/50",
                ].join(" ")}
                onClick={() => {
                  setPlanChoice(choice.value);
                  if (choice.value !== "date") setPlanDate("");
                }}
              >
                {choice.label}
              </button>
            ))}
            <button
              type="button"
              role="radio"
              aria-checked={planChoice === "date"}
              aria-label="Pick a specific date"
              title="Pick a specific date"
              className={[
                "flex items-center justify-center px-3 py-2 min-h-[44px] transition-colors",
                planChoice === "date"
                  ? "bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                  : "bg-white text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700/50",
              ].join(" ")}
              onClick={() => {
                setPlanChoice("date");
              }}
            >
              📅
            </button>
          </div>
          {planChoice === "date" && (
            <div className="mt-2">
              <label htmlFor="plan-date" className="sr-only">
                Pick a date
              </label>
              <input
                id="plan-date"
                type="date"
                aria-label="Pick a date"
                value={planDate}
                onChange={(e) => setPlanDate(e.target.value)}
                className="block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 px-3 py-2 text-sm min-h-[44px]"
              />
            </div>
          )}
        </div>
      )}

      <div className="flex gap-2">
        <Button type="submit" loading={loading} className="flex-1">
          {submitLabel}
        </Button>
        {onCancel && (
          <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
            Cancel
          </Button>
        )}
      </div>
    </form>
  );
}

import { useState, type FormEvent } from "react";
import { Field } from "./Field";
import { Button } from "./Button";
import { messages } from "../theme/messages";

interface TaskFormProps {
  onSubmit: (name: string, description: string) => Promise<void>;
  onCancel?: () => void;
  loading?: boolean;
  initialName?: string;
  initialDescription?: string;
  submitLabel?: string;
}

export function TaskForm({
  onSubmit,
  onCancel,
  loading = false,
  initialName = "",
  initialDescription = "",
  submitLabel = "Add task",
}: TaskFormProps) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [nameError, setNameError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.emptyNameValidation);
      return;
    }
    setNameError(null);
    await onSubmit(trimmed, description.trim());
  }

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

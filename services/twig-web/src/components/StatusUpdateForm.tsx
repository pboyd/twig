import { useState, type FormEvent } from "react";
import { Button } from "./Button";
import { messages } from "../theme/messages";

interface StatusUpdateFormProps {
  initialBody?: string;
  submitLabel: string;
  loading?: boolean;
  onSubmit: (body: string) => void;
  onCancel: () => void;
}

export function StatusUpdateForm({
  initialBody = "",
  submitLabel,
  loading = false,
  onSubmit,
  onCancel,
}: StatusUpdateFormProps) {
  const [body, setBody] = useState(initialBody);
  const [error, setError] = useState<string | null>(null);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = body.trim();
    if (!trimmed) {
      setError(messages.emptyStatusValidation);
      return;
    }
    setError(null);
    onSubmit(trimmed);
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2">
      <textarea
        value={body}
        onChange={(e) => {
          setBody(e.target.value);
          if (error) setError(null);
        }}
        rows={3}
        className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:placeholder-gray-500"
        placeholder="What's the latest?"
      />
      {error && (
        <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
      )}
      <div className="flex gap-2">
        <Button type="submit" loading={loading} className="flex-1">{submitLabel}</Button>
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancel
        </Button>
      </div>
    </form>
  );
}

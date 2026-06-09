export type ToastTone = "success" | "error";

interface ToastProps {
  message: string;
  tone?: ToastTone;
  onDismiss: () => void;
}

export function Toast({ message, tone = "success", onDismiss }: ToastProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      onClick={onDismiss}
      className={[
        "rounded-lg px-4 py-3 text-sm font-medium shadow-lg cursor-pointer",
        tone === "success"
          ? "bg-green-600 text-white dark:bg-green-400 dark:text-gray-900"
          : "bg-red-600 text-white dark:bg-red-400 dark:text-gray-900",
      ].join(" ")}
      data-testid="toast"
    >
      {message}
    </div>
  );
}

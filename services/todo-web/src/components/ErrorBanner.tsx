import { messages } from "../theme/messages";
import { Button } from "./Button";

interface ErrorBannerProps {
  message?: string;
  onRetry?: () => void;
}

export function ErrorBanner({ message = messages.connectivityError, onRetry }: ErrorBannerProps) {
  return (
    <div
      role="alert"
      className="flex items-center justify-between gap-3 rounded-md bg-red-50 px-4 py-3 text-sm text-red-800 dark:bg-red-900/30 dark:text-red-300"
    >
      <span>{message}</span>
      {onRetry && (
        <Button variant="secondary" onClick={onRetry} className="shrink-0 text-xs px-3">
          Try again
        </Button>
      )}
    </div>
  );
}

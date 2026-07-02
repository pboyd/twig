interface CompletionToggleProps {
  completed: boolean;
  disabled?: boolean;
  onToggle: () => void;
}

export function CompletionToggle({ completed, disabled, onToggle }: CompletionToggleProps) {
  return (
    <button
      aria-label={completed ? "Mark incomplete" : "Mark complete"}
      disabled={disabled}
      className={[
        "flex h-11 w-11 shrink-0 items-center justify-center rounded",
        disabled ? "opacity-50" : "",
      ].join(" ")}
      onClick={onToggle}
    >
      <span
        className={[
          "flex h-5 w-5 items-center justify-center rounded-full border-2",
          completed
            ? "border-green-500 bg-green-500 text-white"
            : "border-gray-400 dark:border-gray-500",
        ].join(" ")}
      >
        {completed && (
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
  );
}

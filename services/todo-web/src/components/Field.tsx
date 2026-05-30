import type { InputHTMLAttributes, TextareaHTMLAttributes } from "react";

interface FieldBaseProps {
  label: string;
  error?: string;
  id: string;
}

interface InputFieldProps
  extends FieldBaseProps,
    Omit<InputHTMLAttributes<HTMLInputElement>, "id"> {
  multiline?: false;
}

interface TextareaFieldProps
  extends FieldBaseProps,
    Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, "id"> {
  multiline: true;
}

type FieldProps = InputFieldProps | TextareaFieldProps;

export function Field(props: FieldProps) {
  const { label, error, id, multiline, ...rest } = props;
  const inputClasses = [
    "block w-full rounded-md border px-3 py-2 text-sm",
    "min-h-[44px]",
    "border-gray-300 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none",
    "dark:bg-gray-800 dark:border-gray-600 dark:text-gray-100 dark:focus:border-blue-400",
    error ? "border-red-500 focus:border-red-500 focus:ring-red-500" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      {multiline ? (
        <textarea
          id={id}
          className={inputClasses}
          rows={3}
          {...(rest as TextareaHTMLAttributes<HTMLTextAreaElement>)}
        />
      ) : (
        <input
          id={id}
          className={inputClasses}
          {...(rest as InputHTMLAttributes<HTMLInputElement>)}
        />
      )}
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

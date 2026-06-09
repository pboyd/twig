import { createContext, useCallback, useContext, useRef, useState } from "react";
import { Toast, type ToastTone } from "../components/Toast";

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

interface ToastContextValue {
  show: (message: string, tone?: ToastTone) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx;
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const show = useCallback((message: string, tone: ToastTone = "success") => {
    const id = ++nextId.current;
    setToasts((prev) => [...prev, { id, message, tone }]);
    setTimeout(() => dismiss(id), 3000);
  }, [dismiss]);

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      {toasts.length > 0 && (
        <div
          className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 items-center"
          aria-label="Notifications"
        >
          {toasts.map((t) => (
            <Toast
              key={t.id}
              message={t.message}
              tone={t.tone}
              onDismiss={() => dismiss(t.id)}
            />
          ))}
        </div>
      )}
    </ToastContext.Provider>
  );
}

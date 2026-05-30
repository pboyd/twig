import { useNavigate } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "./Button";

export function AppHeader() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function handleSignOut() {
    try {
      await fetch("/auth/logout", { method: "POST", credentials: "include" });
    } finally {
      queryClient.clear();
      navigate("/login", { replace: true });
    }
  }

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3 dark:border-gray-700 dark:bg-gray-800">
      <span className="text-base font-semibold text-gray-900 dark:text-gray-100">
        Todo
      </span>
      <Button variant="secondary" onClick={handleSignOut} className="text-sm px-3">
        Sign out
      </Button>
    </header>
  );
}

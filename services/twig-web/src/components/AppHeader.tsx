import { useNavigate, NavLink } from "react-router";
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

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    [
      "text-sm font-medium px-3 py-1 rounded-md transition-colors",
      isActive
        ? "bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100"
        : "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100",
    ].join(" ");

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3 dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center gap-1">
        <img src="/logo.svg" alt="" className="h-6 w-6" />
        <span className="text-base font-semibold text-gray-900 dark:text-gray-100 mr-3">
          Twig
        </span>
        <NavLink to="/tasks" className={navLinkClass}>
          Tasks
        </NavLink>
        <NavLink to="/plan" className={navLinkClass}>
          Plan
        </NavLink>
        <NavLink to="/download" className={navLinkClass}>
          Download
        </NavLink>
        <NavLink to="/account" className={navLinkClass}>
          Account
        </NavLink>
      </div>
      <Button variant="secondary" onClick={handleSignOut} className="text-sm px-3">
        Sign out
      </Button>
    </header>
  );
}

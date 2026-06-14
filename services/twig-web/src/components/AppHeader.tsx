import { NavLink } from "react-router";
import { HeaderMenu } from "./HeaderMenu";

export function AppHeader() {
  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    [
      "text-sm font-medium px-3 py-2 rounded-md transition-colors block",
      isActive
        ? "bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100"
        : "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100",
    ].join(" ");

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3 dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center gap-1">
        <NavLink
          to="/tasks"
          aria-label="Twig home"
          className="flex items-center gap-1 mr-1"
        >
          <img src="/logo.svg" alt="" className="h-6 w-6" />
          <span className="hidden sm:inline text-base font-semibold text-gray-900 dark:text-gray-100">
            Twig
          </span>
        </NavLink>

        <NavLink to="/tasks" className={navLinkClass}>
          Tasks
        </NavLink>
        <NavLink to="/plan" className={navLinkClass}>
          Plan
        </NavLink>

        {/* Download and Account: wide viewports only */}
        <NavLink
          to="/download"
          className={(state) => `${navLinkClass(state)} hidden sm:block`}
        >
          Download
        </NavLink>
        <NavLink
          to="/account"
          className={(state) => `${navLinkClass(state)} hidden sm:block`}
        >
          Account
        </NavLink>
      </div>

      <HeaderMenu navLinkClass={navLinkClass} />
    </header>
  );
}

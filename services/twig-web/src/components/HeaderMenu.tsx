import { useEffect, useRef, useState } from "react";
import { NavLink, useLocation } from "react-router";
import { useSignOut } from "../lib/useSignOut";
import { messages } from "../theme/messages";

export function HeaderMenu() {
  const [open, setOpen] = useState(false);
  const signOut = useSignOut();
  const location = useLocation();
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Close on route change
  useEffect(() => {
    setOpen(false);
  }, [location.pathname]);

  // Close on outside pointer-down and Escape
  useEffect(() => {
    if (!open) return;

    function handlePointerDown(e: PointerEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    }

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  async function handleSignOut() {
    setOpen(false);
    await signOut();
  }

  const menuItemClass = ({ isActive }: { isActive: boolean }) =>
    [
      "block w-full text-left px-4 py-2 text-sm font-medium transition-colors",
      isActive
        ? "bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100"
        : "text-gray-600 hover:text-gray-900 hover:bg-gray-50 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-gray-700",
    ].join(" ");

  return (
    <div ref={containerRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        aria-expanded={open}
        aria-controls="header-menu"
        aria-label={messages.menuTriggerLabel}
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center justify-center rounded-md min-h-[44px] min-w-[44px] text-gray-600 hover:text-gray-900 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-gray-700 transition-colors"
      >
        <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <rect x="2" y="4" width="16" height="2" rx="1" />
          <rect x="2" y="9" width="16" height="2" rx="1" />
          <rect x="2" y="14" width="16" height="2" rx="1" />
        </svg>
      </button>

      {open && (
        <div
          id="header-menu"
          className="absolute right-0 mt-1 w-44 rounded-md border border-gray-200 bg-white shadow-md dark:border-gray-700 dark:bg-gray-800 z-50"
        >
          <NavLink
            to="/goals"
            className={menuItemClass}
            onClick={() => setOpen(false)}
          >
            Goals
          </NavLink>
          <NavLink
            to="/account"
            className={menuItemClass}
            onClick={() => setOpen(false)}
          >
            Account
          </NavLink>
          <NavLink
            to="/download"
            className={menuItemClass}
            onClick={() => setOpen(false)}
          >
            Download
          </NavLink>
          <button
            onClick={handleSignOut}
            className={menuItemClass({ isActive: false })}
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  );
}

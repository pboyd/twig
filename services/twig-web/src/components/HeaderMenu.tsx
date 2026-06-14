import { useEffect, useRef, useState } from "react";
import { NavLink, useNavigate, useLocation } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import { messages } from "../theme/messages";

interface HeaderMenuProps {
  navLinkClass: (props: { isActive: boolean }) => string;
}

export function HeaderMenu({ navLinkClass }: HeaderMenuProps) {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const location = useLocation();
  const containerRef = useRef<HTMLDivElement>(null);

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
      if (e.key === "Escape") setOpen(false);
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
    try {
      await fetch("/auth/logout", { method: "POST", credentials: "include" });
    } finally {
      queryClient.clear();
      navigate("/login", { replace: true });
    }
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
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
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
          role="menu"
          className="absolute right-0 mt-1 w-44 rounded-md border border-gray-200 bg-white shadow-md dark:border-gray-700 dark:bg-gray-800 z-50"
        >
          <NavLink
            to="/account"
            role="menuitem"
            className={navLinkClass}
            onClick={() => setOpen(false)}
          >
            Account
          </NavLink>
          <NavLink
            to="/download"
            role="menuitem"
            className={navLinkClass}
            onClick={() => setOpen(false)}
          >
            Download
          </NavLink>
          <button
            role="menuitem"
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

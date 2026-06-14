import { useNavigate } from "react-router";
import { useQueryClient } from "@tanstack/react-query";

/** Shared sign-out action — POST /auth/logout, clear cache, redirect to /login. */
export function useSignOut() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return async function signOut() {
    try {
      await fetch("/auth/logout", { method: "POST", credentials: "include" });
    } finally {
      queryClient.clear();
      navigate("/login", { replace: true });
    }
  };
}

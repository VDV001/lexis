import { create } from "zustand";
import type { User } from "@/types";
import api from "@/lib/api";
import { tryRefreshToken } from "@/lib/auth";

interface SessionState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isRestoring: boolean;
  setAuth: (user: User, accessToken: string) => void;
  clearSession: () => void;
  tryRestore: () => Promise<boolean>;
}

export const useSessionStore = create<SessionState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isRestoring: true,
  setAuth: (user, accessToken) => {
    sessionStorage.setItem("access_token", accessToken);
    set({ user, accessToken, isAuthenticated: true, isRestoring: false });
  },
  clearSession: () => {
    sessionStorage.removeItem("access_token");
    set({ user: null, accessToken: null, isAuthenticated: false, isRestoring: false });
  },
  tryRestore: async () => {
    // Case 1: access_token still in sessionStorage (same-tab reload)
    const token = sessionStorage.getItem("access_token");
    if (token) {
      try {
        const user = await api.get<User>("/users/me");
        set({ user, accessToken: token, isAuthenticated: true, isRestoring: false });
        return true;
      } catch {
        sessionStorage.removeItem("access_token");
      }
    }

    // Case 2: try refresh via HTTP-only cookie (new tab / expired token)
    try {
      const refreshed = await tryRefreshToken();
      if (refreshed) {
        const newToken = sessionStorage.getItem("access_token")!;
        const user = await api.get<User>("/users/me");
        set({ user, accessToken: newToken, isAuthenticated: true, isRestoring: false });
        return true;
      }
    } catch {
      // refresh failed
    }

    set({ isRestoring: false });
    return false;
  },
}));
